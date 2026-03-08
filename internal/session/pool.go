package session

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// UpdateHandler is called for every TDLib update received by any session.
type UpdateHandler func(sessionID string, update interface{})

// Hook receives lifecycle events from the pool.
// All methods must be non-blocking.
type Hook interface {
	OnAdded(sessType Type)
	OnRemoved(sessType Type, finalStatus Status)
	OnStatusChanged(sessType Type, old, new Status)
	OnError(sessType Type)
}

// TDClient is the interface the pool uses to interact with a single TDLib instance.
// The concrete implementation lives in internal/tdlib; this interface keeps the
// session package free of CGO dependencies.
type TDClient interface {
	ID() string
	Close()
	RunEventLoop(ctx context.Context, handler func(update interface{}))
	SendCode(code string) error
	SendPassword(password string) error
	AuthState() string // "waiting_phone", "waiting_code", "waiting_password", "ready", "error", etc.
}

// ClientFactory creates a TDClient for a given session config.
type ClientFactory func(id, phone, token string) (TDClient, error)

// Pool manages a set of concurrent TDLib sessions.
type Pool struct {
	mu       sync.RWMutex
	sessions map[string]*entry

	factory  ClientFactory
	onUpdate UpdateHandler
	hook     Hook
	log      *zap.Logger
}

type entry struct {
	session *Session
	client  TDClient
}

// NewPool creates an empty session pool.
// factory is called each time a new session is added; it must return a ready TDClient.
// hook is optional (pass nil to disable).
func NewPool(factory ClientFactory, onUpdate UpdateHandler, log *zap.Logger, hook Hook) *Pool {
	return &Pool{
		sessions: make(map[string]*entry),
		factory:  factory,
		onUpdate: onUpdate,
		hook:     hook,
		log:      log,
	}
}

// Add registers and starts a user-account session.
func (p *Pool) Add(ctx context.Context, id, phone string) error {
	return p.add(ctx, id, phone, "", TypeAccount)
}

// AddBot registers and starts a bot session.
func (p *Pool) AddBot(ctx context.Context, id, token string) error {
	return p.add(ctx, id, "", token, TypeBot)
}

func (p *Pool) add(ctx context.Context, id, phone, token string, t Type) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.sessions[id]; exists {
		return fmt.Errorf("session %q already exists", id)
	}

	sess := newSession(id, t)
	sess.setStatus(StatusAuthorizing)

	c, err := p.factory(id, phone, token)
	if err != nil {
		sess.setStatus(StatusError)
		p.notify(func(h Hook) { h.OnError(t) })
		return fmt.Errorf("init tdlib for session %q: %w", id, err)
	}

	sessCtx, cancel := context.WithCancel(ctx)
	sess.cancel = cancel

	old := sess.Status
	sess.setStatus(StatusReady)
	p.notify(func(h Hook) {
		h.OnAdded(t)
		h.OnStatusChanged(t, old, StatusReady)
	})

	e := &entry{session: sess, client: c}
	p.sessions[id] = e

	go p.runLoop(sessCtx, e)

	p.log.Info("session added", zap.String("id", id), zap.String("type", string(t)))
	return nil
}

// Remove stops and removes a session from the pool.
func (p *Pool) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}

	finalStatus := e.session.Status
	sessType := e.session.Type

	e.session.cancel()
	e.client.Close()
	delete(p.sessions, id)

	p.notify(func(h Hook) { h.OnRemoved(sessType, finalStatus) })

	p.log.Info("session removed", zap.String("id", id))
	return nil
}

// Get returns a snapshot of a session's state.
func (p *Pool) Get(id string) (*Session, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.sessions[id]
	if !ok {
		return nil, false
	}
	cp := *e.session
	return &cp, true
}

// List returns snapshots of all sessions.
func (p *Pool) List() []*Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]*Session, 0, len(p.sessions))
	for _, e := range p.sessions {
		cp := *e.session
		result = append(result, &cp)
	}
	return result
}

// SendAuthCode sends auth code to the session's TDLib client.
func (p *Pool) SendAuthCode(id, code string) error {
	p.mu.RLock()
	e, ok := p.sessions[id]
	p.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	return e.client.SendCode(code)
}

// SendPassword sends 2FA password to the session's TDLib client.
func (p *Pool) SendPassword(id, password string) error {
	p.mu.RLock()
	e, ok := p.sessions[id]
	p.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	return e.client.SendPassword(password)
}

// GetAuthState returns the auth state for a session.
func (p *Pool) GetAuthState(id string) (string, error) {
	p.mu.RLock()
	e, ok := p.sessions[id]
	p.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("session %q not found", id)
	}
	return e.client.AuthState(), nil
}

// SetUpdateHandler replaces the update handler at runtime.
func (p *Pool) SetUpdateHandler(h UpdateHandler) {
	p.mu.Lock()
	p.onUpdate = h
	p.mu.Unlock()
}

// Len returns the number of active sessions.
func (p *Pool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}

// runLoop drives the TDLib event loop for a single session.
func (p *Pool) runLoop(ctx context.Context, e *entry) {
	log := p.log.With(zap.String("session_id", e.session.ID))
	log.Debug("event loop started")

	e.client.RunEventLoop(ctx, func(update interface{}) {
		p.mu.RLock()
		h := p.onUpdate
		p.mu.RUnlock()
		if h != nil {
			h(e.session.ID, update)
		}
	})

	// Mark disconnected if loop exits without an explicit Remove.
	p.mu.Lock()
	if existing, ok := p.sessions[e.session.ID]; ok && existing == e {
		old := e.session.Status
		e.session.setStatus(StatusDisconnected)
		p.notify(func(h Hook) { h.OnStatusChanged(e.session.Type, old, StatusDisconnected) })
	}
	p.mu.Unlock()

	log.Debug("event loop stopped")
}

func (p *Pool) notify(fn func(Hook)) {
	if p.hook != nil {
		fn(p.hook)
	}
}
