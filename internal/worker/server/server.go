// Package server implements the WorkerService gRPC server.
// It runs on worker nodes and is called by the main node to manage sessions
// and receive a stream of Telegram updates.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	pb "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	"github.com/tgplane/tgplane/internal/metrics"
	"github.com/tgplane/tgplane/internal/session"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WorkerServer implements pb.WorkerServiceServer.
type WorkerServer struct {
	pb.UnimplementedWorkerServiceServer

	id   string
	pool *session.Pool
	log  *zap.Logger

	// updateBus fans out updates to all active Subscribe streams.
	mu          sync.RWMutex
	subscribers map[string]chan *pb.TelegramUpdate

	// counters for metrics
	updatesTotal int64
}

func New(workerID string, pool *session.Pool, log *zap.Logger) *WorkerServer {
	s := &WorkerServer{
		id:          workerID,
		pool:        pool,
		log:         log,
		subscribers: make(map[string]chan *pb.TelegramUpdate),
	}
	// Wire pool updates → all subscribers
	pool.SetUpdateHandler(s.dispatch)
	return s
}

// dispatch is called by the session pool for every incoming TDLib update.
func (s *WorkerServer) dispatch(sessionID string, raw interface{}) {
	payload, err := json.Marshal(raw)
	if err != nil {
		s.log.Warn("marshal update", zap.Error(err))
		return
	}

	upd := &pb.TelegramUpdate{
		SessionId:  sessionID,
		Type:       fmt.Sprintf("%T", raw),
		Payload:    payload,
		ReceivedAt: time.Now().UnixMilli(),
	}

	metrics.UpdatesReceived.WithLabelValues(s.id, "unknown").Inc()

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- upd:
			metrics.UpdatesDispatched.WithLabelValues(s.id).Inc()
		default:
			// subscriber is slow — drop rather than block the update bus
			metrics.UpdatesDropped.WithLabelValues(s.id).Inc()
		}
	}

	s.updatesTotal++
}

// ---- WorkerServiceServer implementation ----

func (s *WorkerServer) Subscribe(req *pb.SubscribeRequest, stream pb.WorkerService_SubscribeServer) error {
	subID := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	ch := make(chan *pb.TelegramUpdate, 256)

	s.mu.Lock()
	s.subscribers[subID] = ch
	s.mu.Unlock()
	metrics.WorkerSubscribers.Inc()

	defer func() {
		s.mu.Lock()
		delete(s.subscribers, subID)
		s.mu.Unlock()
		close(ch)
		metrics.WorkerSubscribers.Dec()
	}()

	filter := make(map[string]struct{}, len(req.SessionIds))
	for _, id := range req.SessionIds {
		filter[id] = struct{}{}
	}

	s.log.Info("subscribe stream opened", zap.String("sub_id", subID))

	for {
		select {
		case <-stream.Context().Done():
			s.log.Info("subscribe stream closed", zap.String("sub_id", subID))
			return nil
		case upd, ok := <-ch:
			if !ok {
				return nil
			}
			if len(filter) > 0 {
				if _, want := filter[upd.SessionId]; !want {
					continue
				}
			}
			if err := stream.Send(upd); err != nil {
				return err
			}
		}
	}
}

func (s *WorkerServer) AddAccount(ctx context.Context, req *pb.AddAccountRequest) (*pb.SessionInfo, error) {
	if err := s.pool.Add(ctx, req.SessionId, req.Phone); err != nil {
		return nil, status.Errorf(codes.Internal, "add account: %v", err)
	}
	return s.sessionInfo(req.SessionId), nil
}

func (s *WorkerServer) AddBot(ctx context.Context, req *pb.AddBotRequest) (*pb.SessionInfo, error) {
	if err := s.pool.AddBot(ctx, req.SessionId, req.Token); err != nil {
		return nil, status.Errorf(codes.Internal, "add bot: %v", err)
	}
	return s.sessionInfo(req.SessionId), nil
}

func (s *WorkerServer) RemoveSession(ctx context.Context, req *pb.RemoveSessionRequest) (*pb.RemoveSessionResponse, error) {
	if err := s.pool.Remove(req.SessionId); err != nil {
		return nil, status.Errorf(codes.NotFound, "remove session: %v", err)
	}
	return &pb.RemoveSessionResponse{SessionId: req.SessionId}, nil
}

func (s *WorkerServer) ListSessions(_ context.Context, _ *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	sessions := s.pool.List()
	infos := make([]*pb.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		infos = append(infos, &pb.SessionInfo{
			SessionId: sess.ID,
			Status:    string(sess.Status),
			Type:      string(sess.Type),
		})
	}
	return &pb.ListSessionsResponse{Sessions: infos}, nil
}

func (s *WorkerServer) GetMetrics(_ context.Context, _ *pb.GetMetricsRequest) (*pb.WorkerMetrics, error) {
	sessions := s.pool.List()
	var ready, errCount int32
	for _, sess := range sessions {
		switch sess.Status {
		case session.StatusReady:
			ready++
		case session.StatusError:
			errCount++
		}
	}
	return &pb.WorkerMetrics{
		WorkerId:     s.id,
		SessionCount: int32(len(sessions)),
		ReadyCount:   ready,
		ErrorCount:   errCount,
		UpdatesTotal: s.updatesTotal,
		CollectedAt:  time.Now().UnixMilli(),
	}, nil
}

func (s *WorkerServer) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true, Version: "0.1.0"}, nil
}

func (s *WorkerServer) sessionInfo(id string) *pb.SessionInfo {
	sess, ok := s.pool.Get(id)
	if !ok {
		return &pb.SessionInfo{SessionId: id, Status: "unknown"}
	}
	return &pb.SessionInfo{
		SessionId: sess.ID,
		Status:    string(sess.Status),
		Type:      string(sess.Type),
	}
}
