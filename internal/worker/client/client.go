// Package client provides a gRPC client for communicating with a worker node.
package client

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// UpdateHandler is called for each update received from the worker stream.
type UpdateHandler func(workerID string, update *pb.TelegramUpdate)

// Client wraps a gRPC connection to a single worker node.
type Client struct {
	id   string
	addr string
	conn *grpc.ClientConn
	rpc  pb.WorkerServiceClient
	log  *zap.Logger
}

// New dials the worker at addr and returns a ready Client.
func New(id, addr string, log *zap.Logger) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial worker %s at %s: %w", id, addr, err)
	}
	return &Client{
		id:   id,
		addr: addr,
		conn: conn,
		rpc:  pb.NewWorkerServiceClient(conn),
		log:  log.With(zap.String("worker_id", id)),
	}, nil
}

// ID returns the worker identifier.
func (c *Client) ID() string { return c.id }

// Close tears down the gRPC connection.
func (c *Client) Close() error { return c.conn.Close() }

// Health checks if the worker is alive.
func (c *Client) Health(ctx context.Context) error {
	resp, err := c.rpc.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("worker %s reports unhealthy", c.id)
	}
	return nil
}

// AddAccount tells the worker to start a user-account session.
func (c *Client) AddAccount(ctx context.Context, sessionID, phone string) (*pb.SessionInfo, error) {
	return c.rpc.AddAccount(ctx, &pb.AddAccountRequest{
		SessionId: sessionID,
		Phone:     phone,
	})
}

// AddBot tells the worker to start a bot session.
func (c *Client) AddBot(ctx context.Context, sessionID, token string) (*pb.SessionInfo, error) {
	return c.rpc.AddBot(ctx, &pb.AddBotRequest{
		SessionId: sessionID,
		Token:     token,
	})
}

// RemoveSession tells the worker to stop a session.
func (c *Client) RemoveSession(ctx context.Context, sessionID string) error {
	_, err := c.rpc.RemoveSession(ctx, &pb.RemoveSessionRequest{SessionId: sessionID})
	return err
}

// ListSessions returns all sessions on this worker.
func (c *Client) ListSessions(ctx context.Context) ([]*pb.SessionInfo, error) {
	resp, err := c.rpc.ListSessions(ctx, &pb.ListSessionsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Sessions, nil
}

// GetMetrics fetches current worker metrics.
func (c *Client) GetMetrics(ctx context.Context) (*pb.WorkerMetrics, error) {
	return c.rpc.GetMetrics(ctx, &pb.GetMetricsRequest{})
}

// Subscribe opens a streaming subscription to updates from this worker.
// It blocks, calling handler for each update, until ctx is cancelled or
// the stream ends. On stream error it returns the error for the caller to retry.
func (c *Client) Subscribe(ctx context.Context, sessionIDs []string, handler UpdateHandler) error {
	stream, err := c.rpc.Subscribe(ctx, &pb.SubscribeRequest{SessionIds: sessionIDs})
	if err != nil {
		return fmt.Errorf("subscribe to worker %s: %w", c.id, err)
	}
	c.log.Info("subscribed to worker update stream")

	for {
		upd, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("worker %s stream error: %w", c.id, err)
		}
		handler(c.id, upd)
	}
}
