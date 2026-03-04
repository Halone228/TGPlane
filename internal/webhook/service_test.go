package webhook_test

import (
	"context"
	"testing"

	"github.com/tgplane/tgplane/internal/webhook"
)

func newSvc() *webhook.Service {
	return webhook.NewService(webhook.NewMemoryRepository())
}

func TestService_CreateAndList(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	wh, err := svc.Create(ctx, "https://example.com/hook", "secret", []string{"message"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if wh.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if wh.URL != "https://example.com/hook" {
		t.Errorf("unexpected URL: %s", wh.URL)
	}

	hooks, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(hooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(hooks))
	}
}

func TestService_Delete(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	wh, _ := svc.Create(ctx, "https://example.com/hook", "", nil)
	if err := svc.Delete(ctx, wh.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	hooks, _ := svc.List(ctx)
	if len(hooks) != 0 {
		t.Errorf("expected 0 webhooks after delete, got %d", len(hooks))
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := newSvc()
	if err := svc.Delete(context.Background(), 9999); err == nil {
		t.Error("expected error deleting nonexistent webhook")
	}
}
