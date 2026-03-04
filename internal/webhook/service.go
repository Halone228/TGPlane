package webhook

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, url, secret string, events []string) (*Webhook, error) {
	return s.repo.Create(ctx, url, secret, events)
}

func (s *Service) List(ctx context.Context) ([]*Webhook, error) {
	return s.repo.List(ctx)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
