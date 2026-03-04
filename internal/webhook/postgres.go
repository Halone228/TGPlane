package webhook

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, url, secret string, events []string) (*Webhook, error) {
	const q = `
		INSERT INTO webhooks (url, secret, events)
		VALUES ($1, $2, $3)
		RETURNING id, url, secret, events, created_at`

	var w Webhook
	err := r.db.QueryRowxContext(ctx, q, url, secret, pq.Array(events)).
		StructScan(&w)
	return &w, err
}

func (r *postgresRepository) List(ctx context.Context) ([]*Webhook, error) {
	rows, err := r.db.QueryxContext(ctx,
		`SELECT id, url, secret, events, created_at FROM webhooks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.StructScan(&w); err != nil {
			return nil, err
		}
		out = append(out, &w)
	}
	return out, rows.Err()
}

func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ensure pq is used (imported for pq.Array).
var _ = errors.New
