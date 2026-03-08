// Package replication persists Telegram updates from the Redis Stream to PostgreSQL.
package replication

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/tgplane/tgplane/internal/stream"
	"go.uber.org/zap"
)

// Replicator reads from the Redis Stream and writes each update to the messages table.
type Replicator struct {
	rdb *redis.Client
	db  *sqlx.DB
	log *zap.Logger
}

func New(rdb *redis.Client, db *sqlx.DB, log *zap.Logger) *Replicator {
	return &Replicator{rdb: rdb, db: db, log: log}
}

// Run blocks until ctx is cancelled.
func (r *Replicator) Run(ctx context.Context) {
	lastID := "$"
	r.log.Info("message replicator started")

	for {
		select {
		case <-ctx.Done():
			r.log.Info("message replicator stopped")
			return
		default:
		}

		msgs, err := r.rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{stream.UpdatesStream, lastID},
			Count:   200,
			Block:   time.Second,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if err != redis.Nil {
				r.log.Error("xread error", zap.Error(err))
				time.Sleep(time.Second)
			}
			continue
		}

		for _, xs := range msgs {
			for _, msg := range xs.Messages {
				lastID = msg.ID
				if err := r.insertWithRetry(ctx, msg.Values, msg.ID); err != nil {
					r.log.Error("insert message failed after retries",
						zap.Error(err), zap.String("stream_id", msg.ID))
				}
			}
		}
	}
}

func (r *Replicator) insertWithRetry(ctx context.Context, vals map[string]interface{}, streamID string) error {
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		if err := r.insert(ctx, vals); err != nil {
			lastErr = err
			r.log.Warn("insert message retry",
				zap.Error(err),
				zap.String("stream_id", streamID),
				zap.Int("attempt", attempt+1),
			)
			continue
		}
		return nil
	}
	return lastErr
}

func (r *Replicator) insert(ctx context.Context, vals map[string]interface{}) error {
	sessionID := strVal(vals, "session_id")
	workerID := strVal(vals, "worker_id")
	msgType := strVal(vals, "type")
	payload := strVal(vals, "payload")
	if payload == "" {
		payload = "{}"
	}

	receivedAt := time.Now()
	if ms := int64Val(vals, "received_at"); ms > 0 {
		receivedAt = time.UnixMilli(ms)
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO messages (session_id, worker_id, type, payload, received_at)
		 VALUES ($1, $2, $3, $4::jsonb, $5)`,
		sessionID, workerID, msgType, payload, receivedAt,
	)
	return err
}

func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func int64Val(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case int64:
		return v
	case string:
		var n int64
		fmt.Sscan(v, &n)
		return n
	}
	return 0
}
