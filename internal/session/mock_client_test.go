package session

import (
	"context"
	"sync"
)

// mockClient implements TDClient for tests.
type mockClient struct {
	mu      sync.Mutex
	id      string
	closed  bool
	updates []interface{}
}

func newMockClient(id string) *mockClient {
	return &mockClient{id: id}
}

func (m *mockClient) ID() string { return m.id }

func (m *mockClient) Close() {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
}

func (m *mockClient) RunEventLoop(ctx context.Context, handler func(interface{})) {
	for _, u := range m.updates {
		select {
		case <-ctx.Done():
			return
		default:
			handler(u)
		}
	}
	<-ctx.Done()
}

func (m *mockClient) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}
