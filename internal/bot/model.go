package bot

import "time"

type Status string

const (
	StatusPending      Status = "pending"
	StatusAuthorizing  Status = "authorizing"
	StatusReady        Status = "ready"
	StatusDisconnected Status = "disconnected"
	StatusError        Status = "error"
)

// Bot represents a Telegram bot managed by TGPlane.
type Bot struct {
	ID        int64     `db:"id"         json:"id"`
	Token     string    `db:"token"      json:"token"`
	SessionID string    `db:"session_id" json:"session_id"`
	Status    Status    `db:"status"     json:"status"`
	Username  *string   `db:"username"   json:"username,omitempty"`
	TGUserID  *int64    `db:"tg_user_id" json:"tg_user_id,omitempty"`
	WorkerID  *string   `db:"worker_id"  json:"worker_id,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreateRequest struct {
	Token string `json:"token" binding:"required"`
}

type UpdateProfileRequest struct {
	Username *string
	TGUserID *int64
	Status   Status
}
