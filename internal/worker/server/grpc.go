package server

import (
	"fmt"
	"net"
	"time"

	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// GRPCServer wraps a grpc.Server configured for the worker.
type GRPCServer struct {
	srv *grpc.Server
	log *zap.Logger
}

func NewGRPCServer(handler *WorkerServer, log *zap.Logger) *GRPCServer {
	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              30 * time.Second,
			Timeout:           10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             15 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	pb.RegisterWorkerServiceServer(srv, handler)
	reflection.Register(srv) // enables grpcurl introspection
	return &GRPCServer{srv: srv, log: log}
}

// Serve starts listening on addr and blocks until the server stops.
func (g *GRPCServer) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	g.log.Info("gRPC server listening", zap.String("addr", addr))
	return g.srv.Serve(lis)
}

// Stop gracefully shuts down the gRPC server.
func (g *GRPCServer) Stop() {
	g.srv.GracefulStop()
}
