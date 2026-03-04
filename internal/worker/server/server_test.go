package server_test

import (
	"context"
	"errors"
	"net"
	"testing"

	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	"github.com/tgplane/tgplane/internal/session"
	"github.com/tgplane/tgplane/internal/worker/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// startServer spins up a real gRPC server on a random port and returns a client + cleanup func.
func startServer(t *testing.T, pool *session.Pool) (pb.WorkerServiceClient, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	workerSrv := server.New("test-worker", pool, zap.NewNop())
	grpcSrv := grpc.NewServer()
	pb.RegisterWorkerServiceServer(grpcSrv, workerSrv)

	go grpcSrv.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	cleanup := func() {
		conn.Close()
		grpcSrv.Stop()
	}
	return pb.NewWorkerServiceClient(conn), cleanup
}

func errFactory(_ string, _ string, _ string) (session.TDClient, error) {
	return nil, errors.New("tdlib not available in tests")
}

func newTestPool() *session.Pool {
	return session.NewPool(errFactory, nil, zap.NewNop(), nil)
}

func TestWorkerServer_Health(t *testing.T) {
	client, cleanup := startServer(t, newTestPool())
	defer cleanup()

	resp, err := client.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("Health RPC: %v", err)
	}
	if !resp.Ok {
		t.Errorf("expected ok=true, got false")
	}
	if resp.Version == "" {
		t.Error("expected non-empty version")
	}
}

func TestWorkerServer_ListSessions_Empty(t *testing.T) {
	client, cleanup := startServer(t, newTestPool())
	defer cleanup()

	resp, err := client.ListSessions(context.Background(), &pb.ListSessionsRequest{})
	if err != nil {
		t.Fatalf("ListSessions RPC: %v", err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(resp.Sessions))
	}
}

func TestWorkerServer_AddBot_FactoryError(t *testing.T) {
	client, cleanup := startServer(t, newTestPool())
	defer cleanup()

	// Factory always returns error → AddBot must return gRPC error.
	_, err := client.AddBot(context.Background(), &pb.AddBotRequest{
		SessionId: "bot-1",
		Token:     "123:ABC",
	})
	if err == nil {
		t.Fatal("expected error from AddBot with broken factory")
	}
}

func TestWorkerServer_RemoveSession_NotFound(t *testing.T) {
	client, cleanup := startServer(t, newTestPool())
	defer cleanup()

	_, err := client.RemoveSession(context.Background(), &pb.RemoveSessionRequest{
		SessionId: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected NotFound error")
	}
}

func TestWorkerServer_GetMetrics(t *testing.T) {
	client, cleanup := startServer(t, newTestPool())
	defer cleanup()

	metrics, err := client.GetMetrics(context.Background(), &pb.GetMetricsRequest{})
	if err != nil {
		t.Fatalf("GetMetrics RPC: %v", err)
	}
	if metrics.WorkerId != "test-worker" {
		t.Errorf("unexpected worker ID: %s", metrics.WorkerId)
	}
	if metrics.SessionCount != 0 {
		t.Errorf("expected 0 sessions, got %d", metrics.SessionCount)
	}
}

func TestWorkerServer_Subscribe_OpenAndClose(t *testing.T) {
	client, cleanup := startServer(t, newTestPool())
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Cancel the context — the stream should close cleanly (EOF or cancelled).
	cancel()

	_, recvErr := stream.Recv()
	if recvErr == nil {
		t.Error("expected stream to close after context cancel, got nil error")
	}
}

func TestWorkerServer_Subscribe_FilterBySessions(t *testing.T) {
	client, cleanup := startServer(t, newTestPool())
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe with a session filter — should open without error.
	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{
		SessionIds: []string{"session-42"},
	})
	if err != nil {
		t.Fatalf("Subscribe with filter: %v", err)
	}

	cancel()
	_, _ = stream.Recv() // drain
}
