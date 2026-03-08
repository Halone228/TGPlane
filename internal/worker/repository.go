package worker

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/tgplane/tgplane/internal/worker/manager"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, id, addr string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO workers (id, addr) VALUES ($1, $2)
		 ON CONFLICT (id) DO UPDATE SET addr = $2`,
		id, addr,
	)
	if err != nil {
		return fmt.Errorf("upsert worker %s: %w", id, err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workers WHERE id = $1`, id)
	return err
}

func (r *Repository) List(ctx context.Context) ([]manager.WorkerEntry, error) {
	var rows []struct {
		ID   string `db:"id"`
		Addr string `db:"addr"`
	}
	if err := r.db.SelectContext(ctx, &rows, `SELECT id, addr FROM workers ORDER BY id`); err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	entries := make([]manager.WorkerEntry, len(rows))
	for i, r := range rows {
		entries[i] = manager.WorkerEntry{ID: r.ID, Addr: r.Addr}
	}
	return entries, nil
}
