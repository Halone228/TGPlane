package bot

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type postgresRepo struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, b *Bot) error {
	query := `
		INSERT INTO bots (token, session_id, status)
		VALUES (:token, :session_id, :status)
		RETURNING id, created_at, updated_at`
	rows, err := r.db.NamedQueryContext(ctx, query, b)
	if err != nil {
		return fmt.Errorf("bot create: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return rows.StructScan(b)
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id int64) (*Bot, error) {
	var b Bot
	if err := r.db.GetContext(ctx, &b, `SELECT * FROM bots WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("bot get by id %d: %w", id, err)
	}
	return &b, nil
}

func (r *postgresRepo) GetBySessionID(ctx context.Context, sessionID string) (*Bot, error) {
	var b Bot
	if err := r.db.GetContext(ctx, &b, `SELECT * FROM bots WHERE session_id = $1`, sessionID); err != nil {
		return nil, fmt.Errorf("bot get by session %s: %w", sessionID, err)
	}
	return &b, nil
}

func (r *postgresRepo) List(ctx context.Context, f ListFilter) ([]*Bot, error) {
	if f.Limit == 0 {
		f.Limit = 50
	}
	query := `SELECT * FROM bots WHERE 1=1`
	args := []interface{}{}
	i := 1

	if f.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", i)
		args = append(args, *f.Status)
		i++
	}
	if f.WorkerID != nil {
		query += fmt.Sprintf(" AND worker_id = $%d", i)
		args = append(args, *f.WorkerID)
		i++
	}
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, f.Limit, f.Offset)

	var bots []*Bot
	if err := r.db.SelectContext(ctx, &bots, query, args...); err != nil {
		return nil, fmt.Errorf("bot list: %w", err)
	}
	return bots, nil
}

func (r *postgresRepo) UpdateProfile(ctx context.Context, id int64, req UpdateProfileRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE bots
		SET username = $1, tg_user_id = $2, status = $3, updated_at = NOW()
		WHERE id = $4`,
		req.Username, req.TGUserID, req.Status, id,
	)
	return err
}

func (r *postgresRepo) UpdateStatus(ctx context.Context, id int64, status Status) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE bots SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}

func (r *postgresRepo) UpdateWorkerID(ctx context.Context, sessionID string, workerID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE bots SET worker_id = $1, updated_at = NOW() WHERE session_id = $2`,
		workerID, sessionID,
	)
	return err
}

func (r *postgresRepo) ListByWorkerID(ctx context.Context, workerID string) ([]*Bot, error) {
	var bots []*Bot
	if err := r.db.SelectContext(ctx, &bots,
		`SELECT * FROM bots WHERE worker_id = $1 ORDER BY id`, workerID); err != nil {
		return nil, fmt.Errorf("bot list by worker %s: %w", workerID, err)
	}
	return bots, nil
}

func (r *postgresRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM bots WHERE id = $1`, id)
	return err
}
