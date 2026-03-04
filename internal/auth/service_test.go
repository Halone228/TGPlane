package auth_test

import (
	"context"
	"testing"

	"github.com/tgplane/tgplane/internal/auth"
)

func newSvc(masterKey string) *auth.Service {
	return auth.NewService(auth.NewMemoryRepository(), masterKey)
}

func TestService_CreateAndValidate(t *testing.T) {
	svc := newSvc("")
	ctx := context.Background()

	k, raw, err := svc.Create(ctx, "ci-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if k.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if len(raw) != 64 { // 32 bytes hex-encoded = 64 chars
		t.Errorf("expected raw key length 64, got %d", len(raw))
	}
	if k.KeyPrefix != raw[:8] {
		t.Errorf("key_prefix mismatch: got %s, want %s", k.KeyPrefix, raw[:8])
	}
	if !svc.Validate(ctx, raw) {
		t.Error("valid key rejected")
	}
}

func TestService_Validate_InvalidKey(t *testing.T) {
	svc := newSvc("")
	if svc.Validate(context.Background(), "notakey") {
		t.Error("invalid key accepted")
	}
}

func TestService_MasterKey(t *testing.T) {
	svc := newSvc("supersecret")
	if !svc.Validate(context.Background(), "supersecret") {
		t.Error("master key rejected")
	}
	if svc.Validate(context.Background(), "wrongkey") {
		t.Error("wrong key accepted")
	}
}

func TestService_List(t *testing.T) {
	svc := newSvc("")
	ctx := context.Background()

	_, _, _ = svc.Create(ctx, "k1")
	_, _, _ = svc.Create(ctx, "k2")

	keys, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestService_Delete(t *testing.T) {
	svc := newSvc("")
	ctx := context.Background()

	k, raw, _ := svc.Create(ctx, "temp")
	if err := svc.Delete(ctx, k.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if svc.Validate(ctx, raw) {
		t.Error("deleted key still valid")
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := newSvc("")
	if err := svc.Delete(context.Background(), 9999); err == nil {
		t.Error("expected error deleting nonexistent key")
	}
}
