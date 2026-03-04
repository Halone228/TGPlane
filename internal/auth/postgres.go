package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, name, keyPrefix, keyHash string) (*APIKey, error) {
	const q = `
		INSERT INTO api_keys (name, key_prefix, key_hash)
		VALUES (:name, :key_prefix, :key_hash)
		RETURNING id, name, key_prefix, key_hash, created_at`

	rows, err := r.db.NamedQueryContext(ctx, q, map[string]interface{}{
		"name":       name,
		"key_prefix": keyPrefix,
		"key_hash":   keyHash,
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, errors.New("no row returned from insert")
	}
	var k APIKey
	if err := rows.StructScan(&k); err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *postgresRepository) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	var k APIKey
	err := r.db.GetContext(ctx, &k,
		`SELECT id, name, key_prefix, key_hash, created_at FROM api_keys WHERE key_hash = $1`, keyHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &k, err
}

func (r *postgresRepository) List(ctx context.Context) ([]*APIKey, error) {
	var keys []*APIKey
	err := r.db.SelectContext(ctx, &keys,
		`SELECT id, name, key_prefix, key_hash, created_at FROM api_keys ORDER BY id`)
	return keys, err
}

func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
