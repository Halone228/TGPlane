package manager_test

import (
	"context"
	"net"
	"testing"

	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	"github.com/tgplane/tgplane/internal/session"
	workerserver "github.com/tgplane/tgplane/internal/worker/server"
	"github.com/tgplane/tgplane/internal/worker/manager"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// fakeWorker runs an in-process gRPC worker with a real session pool.
type fakeWorker struct {
	addr string
	pool *session.Pool
}

func startFakeWorker(t *testing.T, id string) *fakeWorker {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	pool := session.NewPool(func(sessID, _, _ string) (session.TDClient, error) {
		return &noopClient{id: sessID}, nil
	}, nil, zap.NewNop(), nil)

	handler := workerserver.New(id, pool, zap.NewNop())
	srv := grpc.NewServer()
	pb.RegisterWorkerServiceServer(srv, handler)
	go srv.Serve(lis) //nolint:errcheck

	t.Cleanup(func() { srv.Stop() })

	return &fakeWorker{addr: lis.Addr().String(), pool: pool}
}

func noopUpdateHandler(_ string, _ *pb.TelegramUpdate) {}

// testCtx returns a context cancelled automatically when the test ends.
func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}

func newManager(t *testing.T) *manager.Manager {
	t.Helper()
	return manager.New(noopUpdateHandler, zap.NewNop(), nil)
}

// --- Tests ---

func TestManager_NoWorkers_ReturnsError(t *testing.T) {
	mgr := newManager(t)
	_, err := mgr.AssignBot(testCtx(t), "sess-1", "tok")
	if err == nil {
		t.Fatal("expected error with no workers")
	}
}

func TestManager_AddWorker_Health(t *testing.T) {
	ctx := testCtx(t)
	w := startFakeWorker(t, "w1")
	mgr := newManager(t)

	if err := mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w1", Addr: w.addr}); err != nil {
		t.Fatalf("AddWorker: %v", err)
	}
	if workers := mgr.Workers(); len(workers) != 1 {
		t.Errorf("expected 1 worker, got %d", len(workers))
	}
}

func TestManager_AddWorker_Duplicate(t *testing.T) {
	ctx := testCtx(t)
	w := startFakeWorker(t, "w1")
	mgr := newManager(t)

	_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w1", Addr: w.addr})
	if err := mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w1", Addr: w.addr}); err == nil {
		t.Fatal("expected error on duplicate worker ID")
	}
}

func TestManager_AddWorker_Unreachable(t *testing.T) {
	mgr := newManager(t)
	err := mgr.AddWorker(testCtx(t), manager.WorkerConfig{
		ID:   "ghost",
		Addr: "127.0.0.1:1", // nothing listening
	})
	if err == nil {
		t.Fatal("expected error for unreachable worker")
	}
}

func TestManager_RemoveWorker(t *testing.T) {
	ctx := testCtx(t)
	w := startFakeWorker(t, "w1")
	mgr := newManager(t)

	_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w1", Addr: w.addr})
	if err := mgr.RemoveWorker("w1"); err != nil {
		t.Fatalf("RemoveWorker: %v", err)
	}
	if len(mgr.Workers()) != 0 {
		t.Error("expected 0 workers after remove")
	}
}

func TestManager_RemoveWorker_NotFound(t *testing.T) {
	mgr := newManager(t)
	if err := mgr.RemoveWorker("ghost"); err == nil {
		t.Fatal("expected error removing nonexistent worker")
	}
}

func TestManager_LeastLoaded_PicksWorkerWithFewerSessions(t *testing.T) {
	ctx := testCtx(t)
	w1 := startFakeWorker(t, "w1")
	w2 := startFakeWorker(t, "w2")
	mgr := newManager(t)

	_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w1", Addr: w1.addr})
	_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w2", Addr: w2.addr})

	// Pre-load w1 with 3 sessions.
	for i := range 3 {
		_ = w1.pool.AddBot(ctx, formatID("w1-bot", i), "tok")
	}

	workerID, err := mgr.AssignBot(ctx, "new-bot", "token:XYZ")
	if err != nil {
		t.Fatalf("AssignBot: %v", err)
	}
	if workerID != "w2" {
		t.Errorf("expected w2 (0 sessions), got %s", workerID)
	}
}

func TestManager_LeastLoaded_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		sessionsOnW1   int
		sessionsOnW2   int
		sessionsOnW3   int
		expectedWorker string
	}{
		{"w2 has fewer", 5, 1, 3, "w2"},
		{"w3 has fewest", 4, 3, 0, "w3"},
		{"w1 has fewest", 0, 2, 2, "w1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testCtx(t)
			w1 := startFakeWorker(t, "w1")
			w2 := startFakeWorker(t, "w2")
			w3 := startFakeWorker(t, "w3")
			mgr := newManager(t)

			_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w1", Addr: w1.addr})
			_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w2", Addr: w2.addr})
			_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w3", Addr: w3.addr})

			addSessions(ctx, w1.pool, "w1", tt.sessionsOnW1)
			addSessions(ctx, w2.pool, "w2", tt.sessionsOnW2)
			addSessions(ctx, w3.pool, "w3", tt.sessionsOnW3)

			workerID, err := mgr.AssignBot(ctx, "target-bot", "tok")
			if err != nil {
				t.Fatalf("AssignBot: %v", err)
			}
			if workerID != tt.expectedWorker {
				t.Errorf("expected %s, got %s", tt.expectedWorker, workerID)
			}
		})
	}
}

func TestManager_CollectMetrics(t *testing.T) {
	ctx := testCtx(t)
	w1 := startFakeWorker(t, "w1")
	w2 := startFakeWorker(t, "w2")
	mgr := newManager(t)

	_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w1", Addr: w1.addr})
	_ = mgr.AddWorker(ctx, manager.WorkerConfig{ID: "w2", Addr: w2.addr})

	metrics := mgr.CollectMetrics(ctx)
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}
}

// --- helpers ---

type noopClient struct{ id string }

func (n *noopClient) ID() string                                            { return n.id }
func (n *noopClient) Close()                                                {}
func (n *noopClient) RunEventLoop(ctx context.Context, _ func(interface{})) { <-ctx.Done() }
func (n *noopClient) SendCode(_ string) error                               { return nil }
func (n *noopClient) SendPassword(_ string) error                           { return nil }
func (n *noopClient) AuthState() string                                     { return "ready" }

func addSessions(ctx context.Context, pool *session.Pool, prefix string, n int) {
	for i := range n {
		_ = pool.AddBot(ctx, formatID(prefix+"-bot", i), "tok")
	}
}

func formatID(prefix string, i int) string {
	return prefix + "-" + string(rune('0'+i))
}

// Verify grpc import is used.
var _ = grpc.NewServer
var _ = insecure.NewCredentials
