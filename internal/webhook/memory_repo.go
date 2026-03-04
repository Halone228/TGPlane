package webhook

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/lib/pq"
)

type MemoryRepository struct {
	mu   sync.RWMutex
	data map[int64]*Webhook
	seq  atomic.Int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{data: make(map[int64]*Webhook)}
}

func (r *MemoryRepository) Create(_ context.Context, url, secret string, events []string) (*Webhook, error) {
	id := r.seq.Add(1)
	w := &Webhook{ID: id, URL: url, Secret: secret, Events: pq.StringArray(events)}
	r.mu.Lock()
	r.data[id] = w
	r.mu.Unlock()
	return w, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]*Webhook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Webhook, 0, len(r.data))
	for _, w := range r.data {
		out = append(out, w)
	}
	return out, nil
}

func (r *MemoryRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return ErrNotFound
	}
	delete(r.data, id)
	return nil
}
