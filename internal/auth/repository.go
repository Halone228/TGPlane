package auth

import "context"

type Repository interface {
	Create(ctx context.Context, name, keyPrefix, keyHash string) (*APIKey, error)
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	List(ctx context.Context) ([]*APIKey, error)
	Delete(ctx context.Context, id int64) error
}
