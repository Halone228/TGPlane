package manager_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	"github.com/tgplane/tgplane/internal/session"
	"github.com/tgplane/tgplane/internal/worker/manager"
	workerserver "github.com/tgplane/tgplane/internal/worker/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func startFakeWorkerB(b *testing.B, id string) string {
	b.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen: %v", err)
	}
	pool := session.NewPool(func(sessID, _, _ string) (session.TDClient, error) {
		return &noopClient{id: sessID}, nil
	}, nil, zap.NewNop(), nil)
	srv := grpc.NewServer()
	pb.RegisterWorkerServiceServer(srv, workerserver.New(id, pool, zap.NewNop()))
	go srv.Serve(lis) //nolint:errcheck
	b.Cleanup(func() { srv.Stop() })
	return lis.Addr().String()
}

func newManagerB(b *testing.B) *manager.Manager {
	b.Helper()
	return manager.New(func(_ string, _ *pb.TelegramUpdate) {}, zap.NewNop(), nil)
}

// BenchmarkManager_LeastLoaded measures worker selection under N connected workers.
func BenchmarkManager_LeastLoaded(b *testing.B) {
	for _, n := range []int{1, 5, 20, 100} {
		b.Run(fmt.Sprintf("workers=%d", n), func(b *testing.B) {
			mgr := newManagerB(b)
			ctx, cancel := context.WithCancel(context.Background())
			b.Cleanup(cancel)

			for i := 0; i < n; i++ {
				addr := startFakeWorkerB(b, fmt.Sprintf("w-%d", i))
				if err := mgr.AddWorker(ctx, manager.WorkerConfig{
					ID:   fmt.Sprintf("w-%d", i),
					Addr: addr,
				}); err != nil {
					b.Fatalf("AddWorker: %v", err)
				}
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					_, _ = mgr.AssignBot(ctx, fmt.Sprintf("probe-%d-%d", b.N, i), "tok")
					i++
				}
			})
		})
	}
}

// BenchmarkManager_AssignBot measures the full assign path (LeastLoaded + gRPC AddSession).
func BenchmarkManager_AssignBot(b *testing.B) {
	mgr := newManagerB(b)
	ctx, cancel := context.WithCancel(context.Background())
	b.Cleanup(cancel)

	for i := 0; i < 3; i++ {
		addr := startFakeWorkerB(b, fmt.Sprintf("w-%d", i))
		if err := mgr.AddWorker(ctx, manager.WorkerConfig{
			ID:   fmt.Sprintf("w-%d", i),
			Addr: addr,
		}); err != nil {
			b.Fatalf("AddWorker: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.AssignBot(ctx, fmt.Sprintf("bot-sess-%d", i), "token:ABC")
	}
}
