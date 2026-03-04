package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/tgplane/tgplane/internal/stream"
	"go.uber.org/zap"
)

// UpdateEvent is the JSON body POSTed to each webhook.
type UpdateEvent struct {
	SessionID  string `json:"session_id"`
	WorkerID   string `json:"worker_id"`
	Type       string `json:"type"`
	Payload    []byte `json:"payload"`
	ReceivedAt int64  `json:"received_at"`
}

// Dispatcher reads from the Redis Stream and fans out to registered webhooks.
type Dispatcher struct {
	rdb  *redis.Client
	repo Repository
	http *http.Client
	log  *zap.Logger
}

func NewDispatcher(rdb *redis.Client, repo Repository, log *zap.Logger) *Dispatcher {
	return &Dispatcher{
		rdb:  rdb,
		repo: repo,
		http: &http.Client{Timeout: 10 * time.Second},
		log:  log,
	}
}

// Run blocks until ctx is cancelled, continuously reading updates and delivering them.
func (d *Dispatcher) Run(ctx context.Context) {
	lastID := "$" // only process updates arriving after startup
	d.log.Info("webhook dispatcher started")

	for {
		select {
		case <-ctx.Done():
			d.log.Info("webhook dispatcher stopped")
			return
		default:
		}

		msgs, err := d.rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{stream.UpdatesStream, lastID},
			Count:   100,
			Block:   time.Second,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// redis.Nil means no messages within block timeout — normal.
			if err != redis.Nil {
				d.log.Error("xread error", zap.Error(err))
				time.Sleep(time.Second)
			}
			continue
		}

		for _, xstream := range msgs {
			for _, msg := range xstream.Messages {
				lastID = msg.ID
				d.deliver(ctx, msg.Values)
			}
		}
	}
}

func (d *Dispatcher) deliver(ctx context.Context, vals map[string]interface{}) {
	ev := extractEvent(vals)
	body, err := json.Marshal(ev)
	if err != nil {
		d.log.Error("marshal update event", zap.Error(err))
		return
	}
	d.DeliverBody(ctx, body, ev.Type)
}

// DeliverBody fans out a pre-serialised event body to all matching webhooks.
// Exported for use in tests.
func (d *Dispatcher) DeliverBody(ctx context.Context, body []byte, evType string) {
	hooks, err := d.repo.List(ctx)
	if err != nil {
		d.log.Error("list webhooks", zap.Error(err))
		return
	}
	for _, wh := range hooks {
		if !matchesFilter(wh.Events, evType) {
			continue
		}
		d.post(ctx, wh, body, evType)
	}
}

func (d *Dispatcher) post(ctx context.Context, wh *Webhook, body []byte, evType string) {
	deliveryID := uuid.New().String()
	sig := signature(wh.Secret, body)

	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
		if err != nil {
			d.log.Error("build request", zap.String("url", wh.URL), zap.Error(err))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-TGPlane-Delivery", deliveryID)
		req.Header.Set("X-TGPlane-Event", evType)
		if sig != "" {
			req.Header.Set("X-TGPlane-Signature", "sha256="+sig)
		}

		resp, err := d.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
	}

	d.log.Warn("webhook delivery failed",
		zap.String("url", wh.URL),
		zap.String("delivery", deliveryID),
		zap.Error(lastErr),
	)
}

// --- helpers ---

func extractEvent(vals map[string]interface{}) UpdateEvent {
	str := func(v interface{}) string {
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	}
	int64v := func(v interface{}) int64 {
		switch t := v.(type) {
		case int64:
			return t
		case string:
			var n int64
			fmt.Sscan(t, &n)
			return n
		}
		return 0
	}
	var payload []byte
	if b, ok := vals["payload"].(string); ok {
		payload = []byte(b)
	}
	return UpdateEvent{
		SessionID:  str(vals["session_id"]),
		WorkerID:   str(vals["worker_id"]),
		Type:       str(vals["type"]),
		Payload:    payload,
		ReceivedAt: int64v(vals["received_at"]),
	}
}

// matchesFilter returns true when the webhook accepts the given event type.
// An empty events list means "accept all".
func matchesFilter(events []string, evType string) bool {
	if len(events) == 0 {
		return true
	}
	for _, e := range events {
		if e == evType {
			return true
		}
	}
	return false
}

func signature(secret string, body []byte) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
