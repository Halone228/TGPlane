package account

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles account business logic.
type Service struct {
	repo Repository
	log  *zap.Logger
}

func NewService(repo Repository, log *zap.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Add registers a new account. The session is not started yet.
func (s *Service) Add(ctx context.Context, req CreateRequest) (*Account, error) {
	a := &Account{
		Phone:     req.Phone,
		SessionID: uuid.NewString(),
		Status:    StatusPending,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("add account: %w", err)
	}
	s.log.Info("account registered", zap.Int64("id", a.ID), zap.String("phone", a.Phone))
	return a, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Account, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetBySessionID(ctx context.Context, sessionID string) (*Account, error) {
	return s.repo.GetBySessionID(ctx, sessionID)
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]*Account, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Remove(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("remove account: %w", err)
	}
	s.log.Info("account removed", zap.Int64("id", id))
	return nil
}
