package session

import (
	"context"
	"time"
)

// Type indicates whether a session belongs to a user account or a bot.
type Type string

const (
	TypeAccount Type = "account"
	TypeBot     Type = "bot"
)

// Status represents the current lifecycle state of a session.
type Status string

const (
	StatusPending      Status = "pending"
	StatusAuthorizing  Status = "authorizing"
	StatusReady        Status = "ready"
	StatusDisconnected Status = "disconnected"
	StatusError        Status = "error"
)

// Session holds the runtime state for one Telegram account or bot.
type Session struct {
	ID        string
	Type      Type
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time

	// cancel stops the session's event loop goroutine.
	cancel context.CancelFunc
}

func newSession(id string, t Type) *Session {
	return &Session{
		ID:        id,
		Type:      t,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (s *Session) setStatus(status Status) {
	s.Status = status
	s.UpdatedAt = time.Now()
}
