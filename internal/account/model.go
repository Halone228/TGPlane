package account

import "time"

// Status mirrors session.Status but is persisted in the database.
type Status string

const (
	StatusPending      Status = "pending"
	StatusAuthorizing  Status = "authorizing"
	StatusReady        Status = "ready"
	StatusDisconnected Status = "disconnected"
	StatusError        Status = "error"
)

// Account represents a Telegram user account managed by TGPlane.
type Account struct {
	ID        int64     `db:"id"         json:"id"`
	Phone     string    `db:"phone"      json:"phone"`
	SessionID string    `db:"session_id" json:"session_id"`
	Status    Status    `db:"status"     json:"status"`
	FirstName *string   `db:"first_name" json:"first_name,omitempty"`
	LastName  *string   `db:"last_name"  json:"last_name,omitempty"`
	Username  *string   `db:"username"   json:"username,omitempty"`
	TGUserID  *int64    `db:"tg_user_id" json:"tg_user_id,omitempty"`
	WorkerID  *string   `db:"worker_id"  json:"worker_id,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// CreateRequest contains fields required to register a new account.
type CreateRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// UpdateProfileRequest updates Telegram profile info after successful auth.
type UpdateProfileRequest struct {
	FirstName *string
	LastName  *string
	Username  *string
	TGUserID  *int64
	Status    Status
}
