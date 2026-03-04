package account

import "context"

// Repository defines persistence operations for accounts.
// Implementations: postgres (internal/account/postgres.go), in-memory for tests.
type Repository interface {
	Create(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, id int64) (*Account, error)
	GetBySessionID(ctx context.Context, sessionID string) (*Account, error)
	List(ctx context.Context, filter ListFilter) ([]*Account, error)
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
