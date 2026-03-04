package webhook

import "context"

type Repository interface {
	Create(ctx context.Context, url, secret string, events []string) (*Webhook, error)
	List(ctx context.Context) ([]*Webhook, error)
	Delete(ctx context.Context, id int64) error
}
