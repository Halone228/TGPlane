package auth

import (
	"context"
	"sync"
	"sync/atomic"
)

type MemoryRepository struct {
	mu   sync.RWMutex
	keys map[int64]*APIKey
	seq  atomic.Int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{keys: make(map[int64]*APIKey)}
}

func (r *MemoryRepository) Create(_ context.Context, name, keyPrefix, keyHash string) (*APIKey, error) {
	id := r.seq.Add(1)
	k := &APIKey{ID: id, Name: name, KeyPrefix: keyPrefix, KeyHash: keyHash}
	r.mu.Lock()
	r.keys[id] = k
	r.mu.Unlock()
	return k, nil
}

func (r *MemoryRepository) GetByHash(_ context.Context, keyHash string) (*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, k := range r.keys {
		if k.KeyHash == keyHash {
			return k, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryRepository) List(_ context.Context) ([]*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*APIKey, 0, len(r.keys))
	for _, k := range r.keys {
		out = append(out, k)
	}
	return out, nil
}

func (r *MemoryRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.keys[id]; !ok {
		return ErrNotFound
	}
	delete(r.keys, id)
	return nil
}
