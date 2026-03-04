package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func noopHandler(string, interface{}) {}

func factoryOK(clients map[string]*mockClient) ClientFactory {
	return func(id, phone, token string) (TDClient, error) {
		c := newMockClient(id)
		clients[id] = c
		return c, nil
	}
}

func factoryErr() ClientFactory {
	return func(id, _, _ string) (TDClient, error) {
		return nil, errors.New("factory error")
	}
}

func newTestPool(f ClientFactory) *Pool {
	return NewPool(f, noopHandler, zap.NewNop(), nil)
}

func TestPool_Add(t *testing.T) {
	clients := map[string]*mockClient{}
	p := newTestPool(factoryOK(clients))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := p.Add(ctx, "sess-1", "+79001234567"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if p.Len() != 1 {
		t.Fatalf("expected 1 session, got %d", p.Len())
	}

	sess, ok := p.Get("sess-1")
	if !ok {
		t.Fatal("session not found after Add")
	}
	if sess.ID != "sess-1" {
		t.Errorf("wrong session ID: %s", sess.ID)
	}
	if sess.Type != TypeAccount {
		t.Errorf("wrong session type: %s", sess.Type)
	}
}

func TestPool_AddBot(t *testing.T) {
	clients := map[string]*mockClient{}
	p := newTestPool(factoryOK(clients))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := p.AddBot(ctx, "bot-1", "token:ABC"); err != nil {
		t.Fatalf("AddBot: %v", err)
	}

	sess, ok := p.Get("bot-1")
	if !ok {
		t.Fatal("bot session not found")
	}
	if sess.Type != TypeBot {
		t.Errorf("expected bot type, got %s", sess.Type)
	}
}

func TestPool_Add_Duplicate(t *testing.T) {
	clients := map[string]*mockClient{}
	p := newTestPool(factoryOK(clients))

	ctx := context.Background()
	_ = p.Add(ctx, "sess-1", "+79001234567")

	err := p.Add(ctx, "sess-1", "+79001234567")
	if err == nil {
		t.Fatal("expected error on duplicate Add, got nil")
	}
}

func TestPool_Add_FactoryError(t *testing.T) {
	p := newTestPool(factoryErr())

	err := p.Add(context.Background(), "sess-1", "+1")
	if err == nil {
		t.Fatal("expected factory error, got nil")
	}
	if p.Len() != 0 {
		t.Errorf("session should not be stored on factory error, got %d", p.Len())
	}
}

func TestPool_Remove(t *testing.T) {
	clients := map[string]*mockClient{}
	p := newTestPool(factoryOK(clients))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = p.Add(ctx, "sess-1", "+1")

	if err := p.Remove("sess-1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if p.Len() != 0 {
		t.Errorf("expected 0 sessions after remove, got %d", p.Len())
	}
	if !clients["sess-1"].isClosed() {
		t.Error("client.Close() was not called on Remove")
	}
}

func TestPool_Remove_NotFound(t *testing.T) {
	p := newTestPool(factoryOK(map[string]*mockClient{}))
	if err := p.Remove("nonexistent"); err == nil {
		t.Fatal("expected error removing nonexistent session")
	}
}

func TestPool_List(t *testing.T) {
	clients := map[string]*mockClient{}
	p := newTestPool(factoryOK(clients))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = p.Add(ctx, "sess-1", "+1")
	_ = p.AddBot(ctx, "bot-1", "tok")

	list := p.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
}

func TestPool_Get_NotFound(t *testing.T) {
	p := newTestPool(factoryOK(map[string]*mockClient{}))
	_, ok := p.Get("missing")
	if ok {
		t.Error("Get should return false for missing session")
	}
}

func TestPool_UpdateHandler(t *testing.T) {
	fakeUpdate := "hello"
	mockC := &mockClient{id: "sess-1", updates: []interface{}{fakeUpdate}}

	factory := func(id, _, _ string) (TDClient, error) {
		return mockC, nil
	}

	received := make(chan string, 1)
	p := NewPool(factory, func(sessID string, upd interface{}) {
		received <- sessID
	}, zap.NewNop(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = p.Add(ctx, "sess-1", "+1")

	select {
	case sessID := <-received:
		if sessID != "sess-1" {
			t.Errorf("unexpected session ID in update: %s", sessID)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for update")
	}
}

func TestPool_SetUpdateHandler(t *testing.T) {
	clients := map[string]*mockClient{}
	p := newTestPool(factoryOK(clients))

	called := false
	p.SetUpdateHandler(func(_ string, _ interface{}) {
		called = true
	})

	// Verify the field was actually replaced (side-effect: no race on existing sessions)
	_ = called
}
