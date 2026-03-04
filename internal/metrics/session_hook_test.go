package metrics_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tgplane/tgplane/internal/metrics"
	"github.com/tgplane/tgplane/internal/session"
	"go.uber.org/zap"
)

// Verify Hook interface is satisfied at compile time.
var _ session.Hook = (*metrics.SessionHook)(nil)

func TestSessionHook_SmokeNoRace(t *testing.T) {
	hook := metrics.NewSessionHook()

	// All calls must be non-blocking and race-free.
	hook.OnAdded(session.TypeBot)
	hook.OnAdded(session.TypeAccount)
	hook.OnStatusChanged(session.TypeBot, session.StatusAuthorizing, session.StatusReady)
	hook.OnStatusChanged(session.TypeAccount, session.StatusAuthorizing, session.StatusReady)
	hook.OnRemoved(session.TypeBot, session.StatusReady)
	hook.OnError(session.TypeAccount)
}

func TestSessionHook_IntegratesWithPool(t *testing.T) {
	hook := metrics.NewSessionHook()

	callCount := 0
	factory := func(id, _, _ string) (session.TDClient, error) {
		callCount++
		return nil, errors.New("no tdlib")
	}

	pool := session.NewPool(factory, nil, zap.NewNop(), hook)
	ctx := context.Background()

	// Factory errors → OnError should be called without panic.
	_ = pool.Add(ctx, "s1", "+1")

	if callCount != 1 {
		t.Errorf("expected factory called once, got %d", callCount)
	}
}
