//go:build integration

// Package testhelper provides utilities for integration tests.
package testhelper

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/tgplane/tgplane/internal/config"
	"github.com/tgplane/tgplane/internal/database"
)

// NewPostgresDB starts a postgres container, runs migrations, and returns a ready *sqlx.DB.
// The container is terminated when the test completes.
func NewPostgresDB(t *testing.T, migrationsDir string) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("tgplane_test"),
		tcpostgres.WithUsername("tgplane"),
		tcpostgres.WithPassword("tgplane"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { container.Terminate(ctx) }) //nolint:errcheck

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := database.Connect(config.DatabaseConfig{
		DSN:          dsn,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	})
	if err != nil {
		t.Fatalf("connect to test postgres: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.Migrate(db, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}
