package bot

import "context"

type Repository interface {
	Create(ctx context.Context, b *Bot) error
	GetByID(ctx context.Context, id int64) (*Bot, error)
	GetBySessionID(ctx context.Context, sessionID string) (*Bot, error)
	List(ctx context.Context, filter ListFilter) ([]*Bot, error)
	UpdateProfile(ctx context.Context, id int64, req UpdateProfileRequest) error
	UpdateStatus(ctx context.Context, id int64, status Status) error
	Delete(ctx context.Context, id int64) error
}

type ListFilter struct {
	Status   *Status
	WorkerID *string
	Limit    int
	Offset   int
}
