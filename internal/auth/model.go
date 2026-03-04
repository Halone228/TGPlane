package auth

import "time"

// APIKey represents a stored API key (hash only — raw key shown once at creation).
type APIKey struct {
	ID        int64     `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	KeyPrefix string    `db:"key_prefix" json:"key_prefix"` // first 8 chars, for UI display
	KeyHash   string    `db:"key_hash"   json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
