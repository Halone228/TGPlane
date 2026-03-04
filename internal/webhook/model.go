package webhook

import (
	"errors"
	"time"

	"github.com/lib/pq"
)

var ErrNotFound = errors.New("webhook not found")

// Webhook represents a registered HTTP callback endpoint.
type Webhook struct {
	ID        int64          `db:"id"         json:"id"`
	URL       string         `db:"url"        json:"url"`
	Secret    string         `db:"secret"     json:"-"`     // not exposed in list responses
	Events    pq.StringArray `db:"events"     json:"events"` // empty = all events
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
}
