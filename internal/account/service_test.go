package account

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
	a, err := svc.Add(context.Background(), CreateRequest{Phone: "+79001234567"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if a.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if a.Phone != "+79001234567" {
		t.Errorf("unexpected phone: %s", a.Phone)
	}
	if a.SessionID == "" {
		t.Error("SessionID must be set")
	}
	if a.Status != StatusPending {
		t.Errorf("expected pending status, got %s", a.Status)
	}
}

func TestService_Add_DuplicatePhone(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Add(ctx, CreateRequest{Phone: "+1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Add(ctx, CreateRequest{Phone: "+1"})
	if err == nil {
		t.Fatal("expected error on duplicate phone")
	}
}

func TestService_Get(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	created, _ := svc.Add(ctx, CreateRequest{Phone: "+1"})
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: want %d, got %d", created.ID, got.ID)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.Get(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for missing account")
	}
}

func TestService_List(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Add(ctx, CreateRequest{Phone: "+1"})
	_, _ = svc.Add(ctx, CreateRequest{Phone: "+2"})

	accounts, err := svc.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestService_List_FilterByStatus(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	a, _ := svc.Add(ctx, CreateRequest{Phone: "+1"})
	_, _ = svc.Add(ctx, CreateRequest{Phone: "+2"})

	// Manually change status of first account via repo
	svc.repo.UpdateStatus(ctx, a.ID, StatusReady)

	status := StatusReady
	accounts, _ := svc.List(ctx, ListFilter{Status: &status})
	if len(accounts) != 1 {
		t.Errorf("expected 1 ready account, got %d", len(accounts))
	}
	if accounts[0].Status != StatusReady {
		t.Errorf("expected ready status, got %s", accounts[0].Status)
	}
}

func TestService_Remove(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	a, _ := svc.Add(ctx, CreateRequest{Phone: "+1"})

	if err := svc.Remove(ctx, a.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, err := svc.Get(ctx, a.ID)
	if err == nil {
		t.Fatal("account should not exist after Remove")
	}
}

func TestService_Remove_NotFound(t *testing.T) {
	svc := newTestService()
	if err := svc.Remove(context.Background(), 9999); err == nil {
		t.Fatal("expected error removing nonexistent account")
	}
}
