// Package stream publishes Telegram updates to a Redis Stream.
package stream

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
)

// UpdatesStream is the Redis stream key for Telegram updates.
const UpdatesStream = "tgplane:updates"

// Publisher writes updates to the Redis Stream.
type Publisher struct {
	rdb *redis.Client
}

func NewPublisher(rdb *redis.Client) *Publisher {
	return &Publisher{rdb: rdb}
}

// Publish adds an update to the stream. Keeps at most ~10 000 entries.
func (p *Publisher) Publish(ctx context.Context, workerID string, upd *pb.TelegramUpdate) error {
	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: UpdatesStream,
		MaxLen: 10_000,
		Approx: true,
		Values: map[string]interface{}{
			"session_id":  upd.SessionId,
			"worker_id":   workerID,
			"type":        upd.Type,
			"payload":     upd.Payload,  // raw bytes stored as Redis bulk string
			"received_at": time.Now().UnixMilli(),
		},
	}).Err()
}
