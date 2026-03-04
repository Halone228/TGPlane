package bot

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	repo Repository
	log  *zap.Logger
}

func NewService(repo Repository, log *zap.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) Add(ctx context.Context, req CreateRequest) (*Bot, error) {
	b := &Bot{
		Token:     req.Token,
		SessionID: uuid.NewString(),
		Status:    StatusPending,
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, fmt.Errorf("add bot: %w", err)
	}
	s.log.Info("bot registered", zap.Int64("id", b.ID))
	return b, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Bot, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetBySessionID(ctx context.Context, sessionID string) (*Bot, error) {
	return s.repo.GetBySessionID(ctx, sessionID)
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]*Bot, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Remove(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("remove bot: %w", err)
	}
	s.log.Info("bot removed", zap.Int64("id", id))
	return nil
}
