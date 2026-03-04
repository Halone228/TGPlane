package account

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
)

// MemoryRepository is an in-memory implementation of Repository.
// Use in unit tests only.
type MemoryRepository struct {
	mu       sync.RWMutex
	store    map[int64]*Account
	byPhone  map[string]int64
	bySession map[string]int64
	seq      atomic.Int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store:     make(map[int64]*Account),
		byPhone:   make(map[string]int64),
		bySession: make(map[string]int64),
	}
}

func (r *MemoryRepository) Create(_ context.Context, a *Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byPhone[a.Phone]; exists {
		return fmt.Errorf("phone %q already exists", a.Phone)
	}
	id := r.seq.Add(1)
	a.ID = id
	cp := *a
	r.store[id] = &cp
	r.byPhone[a.Phone] = id
	r.bySession[a.SessionID] = id
	return nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id int64) (*Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.store[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *a
	return &cp, nil
}

func (r *MemoryRepository) GetBySessionID(_ context.Context, sessionID string) (*Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.bySession[sessionID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *r.store[id]
	return &cp, nil
}

func (r *MemoryRepository) List(_ context.Context, f ListFilter) ([]*Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := f.Limit
	if limit == 0 {
		limit = 50
	}
	result := make([]*Account, 0)
	for _, a := range r.store {
		if f.Status != nil && a.Status != *f.Status {
			continue
		}
		cp := *a
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
	a, ok := r.store[id]
	if !ok {
		return sql.ErrNoRows
	}
	a.FirstName = req.FirstName
	a.LastName = req.LastName
	a.Username = req.Username
	a.TGUserID = req.TGUserID
	a.Status = req.Status
	return nil
}

func (r *MemoryRepository) UpdateStatus(_ context.Context, id int64, status Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.store[id]
	if !ok {
		return sql.ErrNoRows
	}
	a.Status = status
	return nil
}

func (r *MemoryRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.store[id]
	if !ok {
		return sql.ErrNoRows
	}
	delete(r.byPhone, a.Phone)
	delete(r.bySession, a.SessionID)
	delete(r.store, id)
	return nil
}
