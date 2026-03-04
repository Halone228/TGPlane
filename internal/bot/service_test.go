package bot

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func newTestService() *Service {
	return NewService(NewMemoryRepository(), zap.NewNop())
}

func TestService_Add(t *testing.T) {
	svc := newTestService()
	b, err := svc.Add(context.Background(), CreateRequest{Token: "123:ABC"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if b.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if b.Token != "123:ABC" {
		t.Errorf("unexpected token: %s", b.Token)
	}
	if b.SessionID == "" {
		t.Error("SessionID must be set")
	}
	if b.Status != StatusPending {
		t.Errorf("expected pending, got %s", b.Status)
	}
}

func TestService_Add_DuplicateToken(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Add(ctx, CreateRequest{Token: "tok"})
	_, err := svc.Add(ctx, CreateRequest{Token: "tok"})
	if err == nil {
		t.Fatal("expected error on duplicate token")
	}
}

func TestService_Get(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	created, _ := svc.Add(ctx, CreateRequest{Token: "tok"})
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: want %d, got %d", created.ID, got.ID)
	}
}

func TestService_List(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Add(ctx, CreateRequest{Token: "tok1"})
	_, _ = svc.Add(ctx, CreateRequest{Token: "tok2"})

	bots, err := svc.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(bots) != 2 {
		t.Errorf("expected 2 bots, got %d", len(bots))
	}
}

func TestService_Remove(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	b, _ := svc.Add(ctx, CreateRequest{Token: "tok"})
	if err := svc.Remove(ctx, b.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, err := svc.Get(ctx, b.ID)
	if err == nil {
		t.Fatal("bot should not exist after Remove")
	}
}
