package account

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type postgresRepo struct {
	db *sqlx.DB
}

// NewPostgresRepository returns a PostgreSQL-backed Repository.
func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, a *Account) error {
	query := `
		INSERT INTO accounts (phone, session_id, status)
		VALUES (:phone, :session_id, :status)
		RETURNING id, created_at, updated_at`
	rows, err := r.db.NamedQueryContext(ctx, query, a)
	if err != nil {
		return fmt.Errorf("account create: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return rows.StructScan(a)
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	var a Account
	err := r.db.GetContext(ctx, &a, `SELECT * FROM accounts WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("account get by id %d: %w", id, err)
	}
	return &a, nil
}

func (r *postgresRepo) GetBySessionID(ctx context.Context, sessionID string) (*Account, error) {
	var a Account
	err := r.db.GetContext(ctx, &a, `SELECT * FROM accounts WHERE session_id = $1`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("account get by session %s: %w", sessionID, err)
	}
	return &a, nil
}

func (r *postgresRepo) List(ctx context.Context, f ListFilter) ([]*Account, error) {
	if f.Limit == 0 {
		f.Limit = 50
	}
	query := `SELECT * FROM accounts WHERE 1=1`
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

	var accounts []*Account
	if err := r.db.SelectContext(ctx, &accounts, query, args...); err != nil {
		return nil, fmt.Errorf("account list: %w", err)
	}
	return accounts, nil
}

func (r *postgresRepo) UpdateProfile(ctx context.Context, id int64, req UpdateProfileRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE accounts
		SET first_name = $1, last_name = $2, username = $3,
		    tg_user_id = $4, status = $5, updated_at = NOW()
		WHERE id = $6`,
		req.FirstName, req.LastName, req.Username, req.TGUserID, req.Status, id,
	)
	if err != nil {
		return fmt.Errorf("account update profile %d: %w", id, err)
	}
	return nil
}

func (r *postgresRepo) UpdateStatus(ctx context.Context, id int64, status Status) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE accounts SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("account update status %d: %w", id, err)
	}
	return nil
}

func (r *postgresRepo) UpdateWorkerID(ctx context.Context, sessionID string, workerID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET worker_id = $1, updated_at = NOW() WHERE session_id = $2`,
		workerID, sessionID,
	)
	return err
}

func (r *postgresRepo) ListByWorkerID(ctx context.Context, workerID string) ([]*Account, error) {
	var accounts []*Account
	if err := r.db.SelectContext(ctx, &accounts,
		`SELECT * FROM accounts WHERE worker_id = $1 ORDER BY id`, workerID); err != nil {
		return nil, fmt.Errorf("account list by worker %s: %w", workerID, err)
	}
	return accounts, nil
}

func (r *postgresRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("account delete %d: %w", id, err)
	}
	return nil
}
