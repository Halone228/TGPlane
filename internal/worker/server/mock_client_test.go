package server_test

import "context"

// mockClient implements session.TDClient for use in server tests.
type mockClient struct {
	id     string
	update interface{}
	done   chan struct{}
}

func (m *mockClient) ID() string { return m.id }
func (m *mockClient) Close()     {}

func (m *mockClient) RunEventLoop(ctx context.Context, handler func(interface{})) {
	if m.update != nil {
		handler(m.update)
	}
	if m.done != nil {
		select {
		case m.done <- struct{}{}:
		default:
		}
	}
	<-ctx.Done()
}
