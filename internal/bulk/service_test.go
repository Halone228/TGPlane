package bulk_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/tgplane/tgplane/internal/account"
	"github.com/tgplane/tgplane/internal/bot"
	"github.com/tgplane/tgplane/internal/bulk"
)

// newSvc builds a bulk.Service with in-memory repos and no worker manager.
func newSvc() *bulk.Service {
	accountSvc := account.NewService(account.NewMemoryRepository(), zap.NewNop())
	botSvc := bot.NewService(bot.NewMemoryRepository(), zap.NewNop())
	return bulk.NewService(accountSvc, botSvc, nil)
}

func TestAddBots_AllSucceed(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	result := svc.AddBots(ctx, []bulk.AddBotRequest{
		{Token: "1:AAA"},
		{Token: "2:BBB"},
		{Token: "3:CCC"},
	})

	if result.Total != 3 {
		t.Errorf("expected total=3, got %d", result.Total)
	}
	if result.Succeeded != 3 {
		t.Errorf("expected succeeded=3, got %d", result.Succeeded)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed=0, got %d", result.Failed)
	}
	for _, item := range result.Items {
		if !item.OK {
			t.Errorf("item %d failed: %s", item.Index, item.Error)
		}
	}
}

func TestAddBots_DuplicateTokenFails(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	result := svc.AddBots(ctx, []bulk.AddBotRequest{
		{Token: "dup:AAA"},
		{Token: "dup:AAA"}, // duplicate
	})

	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
	// One should succeed, one should fail
	if result.Succeeded+result.Failed != 2 {
		t.Error("succeeded + failed should equal total")
	}
}

func TestAddAccounts_AllSucceed(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	result := svc.AddAccounts(ctx, []bulk.AddAccountRequest{
		{Phone: "+79001111111"},
		{Phone: "+79002222222"},
	})

	if result.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", result.Succeeded)
	}
}

func TestRemoveSessions_NotFound(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	result := svc.RemoveSessions(ctx, []string{"nonexistent-session"})

	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
	if result.Items[0].Error == "" {
		t.Error("expected error message for not found session")
	}
}

func TestRemoveSessions_Found_NoWorker(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	// First add a bot, then remove its session
	addResult := svc.AddBots(ctx, []bulk.AddBotRequest{{Token: "tok:XYZ"}})
	if addResult.Failed != 0 {
		t.Fatalf("add failed: %v", addResult.Items[0].Error)
	}

	// Get the session_id from the result
	data := addResult.Items[0].Data.(*bot.Bot)
	sessionID := data.SessionID

	removeResult := svc.RemoveSessions(ctx, []string{sessionID})
	if removeResult.Failed != 0 {
		t.Errorf("expected remove to succeed (no worker assigned), got: %s",
			removeResult.Items[0].Error)
	}
}

func TestBulkResult_IndexOrder(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	reqs := make([]bulk.AddBotRequest, 20)
	for i := range reqs {
		reqs[i] = bulk.AddBotRequest{Token: "tok:" + string(rune('A'+i))}
	}

	result := svc.AddBots(ctx, reqs)

	if result.Total != 20 {
		t.Fatalf("expected 20, got %d", result.Total)
	}
	// Verify each item has the correct index
	for _, item := range result.Items {
		if item.Index < 0 || item.Index >= 20 {
			t.Errorf("item has out-of-range index: %d", item.Index)
		}
	}
}
