package bot

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tgplane/tgplane/internal/crypto"
	"go.uber.org/zap"
)

type Service struct {
	repo Repository
	log  *zap.Logger
	enc  *crypto.TokenEncryptor
}

func NewService(repo Repository, log *zap.Logger, enc ...*crypto.TokenEncryptor) *Service {
	var e *crypto.TokenEncryptor
	if len(enc) > 0 {
		e = enc[0]
	}
	return &Service{repo: repo, log: log, enc: e}
}

func (s *Service) Add(ctx context.Context, req CreateRequest) (*Bot, error) {
	encToken, err := s.enc.Encrypt(req.Token)
	if err != nil {
		return nil, fmt.Errorf("encrypt token: %w", err)
	}
	b := &Bot{
		Token:     encToken,
		SessionID: uuid.NewString(),
		Status:    StatusPending,
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, fmt.Errorf("add bot: %w", err)
	}
	s.log.Info("bot registered", zap.Int64("id", b.ID))
	// Return the plaintext token to the caller.
	b.Token = req.Token
	return b, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Bot, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.decryptToken(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) GetBySessionID(ctx context.Context, sessionID string) (*Bot, error) {
	b, err := s.repo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.decryptToken(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]*Bot, error) {
	bots, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	for _, b := range bots {
		if err := s.decryptToken(b); err != nil {
			return nil, err
		}
	}
	return bots, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, status Status) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *Service) UpdateWorkerID(ctx context.Context, sessionID, workerID string) error {
	return s.repo.UpdateWorkerID(ctx, sessionID, workerID)
}

func (s *Service) ListByWorkerID(ctx context.Context, workerID string) ([]*Bot, error) {
	bots, err := s.repo.ListByWorkerID(ctx, workerID)
	if err != nil {
		return nil, err
	}
	for _, b := range bots {
		if err := s.decryptToken(b); err != nil {
			return nil, err
		}
	}
	return bots, nil
}

func (s *Service) decryptToken(b *Bot) error {
	plain, err := s.enc.Decrypt(b.Token)
	if err != nil {
		return fmt.Errorf("decrypt token: %w", err)
	}
	b.Token = plain
	return nil
}

func (s *Service) Remove(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("remove bot: %w", err)
	}
	s.log.Info("bot removed", zap.Int64("id", id))
	return nil
}
