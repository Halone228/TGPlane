package bot

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	store     map[int64]*Bot
	byToken   map[string]int64
	bySession map[string]int64
	seq       atomic.Int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store:     make(map[int64]*Bot),
		byToken:   make(map[string]int64),
		bySession: make(map[string]int64),
	}
}

func (r *MemoryRepository) Create(_ context.Context, b *Bot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byToken[b.Token]; exists {
		return fmt.Errorf("token already exists")
	}
	id := r.seq.Add(1)
	b.ID = id
	cp := *b
	r.store[id] = &cp
	r.byToken[b.Token] = id
	r.bySession[b.SessionID] = id
	return nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id int64) (*Bot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.store[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *b
	return &cp, nil
}

func (r *MemoryRepository) GetBySessionID(_ context.Context, sessionID string) (*Bot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.bySession[sessionID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *r.store[id]
	return &cp, nil
}

func (r *MemoryRepository) List(_ context.Context, f ListFilter) ([]*Bot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := f.Limit
	if limit == 0 {
		limit = 50
	}
	result := make([]*Bot, 0)
	for _, b := range r.store {
		if f.Status != nil && b.Status != *f.Status {
			continue
		}
		cp := *b
		result = append(result, &cp)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateProfile(_ context.Context, id int64, req UpdateProfileRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.store[id]
	if !ok {
		return sql.ErrNoRows
	}
	b.Username = req.Username
	b.TGUserID = req.TGUserID
	b.Status = req.Status
	return nil
}

func (r *MemoryRepository) UpdateStatus(_ context.Context, id int64, status Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.store[id]
	if !ok {
		return sql.ErrNoRows
	}
	b.Status = status
	return nil
}

func (r *MemoryRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.store[id]
	if !ok {
		return sql.ErrNoRows
	}
	delete(r.byToken, b.Token)
	delete(r.bySession, b.SessionID)
	delete(r.store, id)
	return nil
}
