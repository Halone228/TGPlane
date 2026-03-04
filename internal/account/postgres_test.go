//go:build integration

package account_test

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/tgplane/tgplane/internal/account"
	"github.com/tgplane/tgplane/internal/config"
	"github.com/tgplane/tgplane/internal/database"
	"go.uber.org/zap"
)

// sharedDB is initialized once in TestMain and reused across all tests.
var sharedDB *sqlx.DB

func TestMain(m *testing.M) {
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
		panic("start postgres container: " + err.Error())
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("connection string: " + err.Error())
	}

	sharedDB, err = database.Connect(config.DatabaseConfig{
		DSN: dsn, MaxOpenConns: 5, MaxIdleConns: 2,
	})
	if err != nil {
		panic("connect: " + err.Error())
	}

	if err := database.Migrate(sharedDB, "../../migrations"); err != nil {
		panic("migrate: " + err.Error())
	}

	code := m.Run()

	sharedDB.Close()
	container.Terminate(ctx) //nolint:errcheck

	os.Exit(code)
}

// cleanDB truncates all tables between tests for isolation.
func cleanDB(t *testing.T) {
	t.Helper()
	_, err := sharedDB.Exec(`TRUNCATE accounts RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("clean DB: %v", err)
	}
}

func newIntegrationService(t *testing.T) *account.Service {
	t.Helper()
	cleanDB(t)
	return account.NewService(account.NewPostgresRepository(sharedDB), zap.NewNop())
}

func TestPostgresRepo_CreateAndGet(t *testing.T) {
	svc := newIntegrationService(t)
	ctx := context.Background()

	a, err := svc.Add(ctx, account.CreateRequest{Phone: "+79001234567"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("expected non-zero ID from postgres")
	}

	got, err := svc.Get(ctx, a.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Phone != "+79001234567" {
		t.Errorf("phone mismatch: got %s", got.Phone)
	}
}

func TestPostgresRepo_List(t *testing.T) {
	svc := newIntegrationService(t)
	ctx := context.Background()

	for _, phone := range []string{"+1", "+2", "+3"} {
		if _, err := svc.Add(ctx, account.CreateRequest{Phone: phone}); err != nil {
			t.Fatalf("Add %s: %v", phone, err)
		}
	}

	accounts, err := svc.List(ctx, account.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(accounts) != 3 {
		t.Errorf("expected 3, got %d", len(accounts))
	}
}

func TestPostgresRepo_UpdateStatus(t *testing.T) {
	svc := newIntegrationService(t)
	repo := account.NewPostgresRepository(sharedDB)
	ctx := context.Background()

	a, err := svc.Add(ctx, account.CreateRequest{Phone: "+1"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := repo.UpdateStatus(ctx, a.ID, account.StatusReady); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := svc.Get(ctx, a.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Status != account.StatusReady {
		t.Errorf("expected ready, got %s", got.Status)
	}
}

func TestPostgresRepo_Delete(t *testing.T) {
	svc := newIntegrationService(t)
	ctx := context.Background()

	a, _ := svc.Add(ctx, account.CreateRequest{Phone: "+1"})
	_ = svc.Remove(ctx, a.ID)

	_, err := svc.Get(ctx, a.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestPostgresRepo_DuplicatePhone(t *testing.T) {
	svc := newIntegrationService(t)
	ctx := context.Background()

	_, _ = svc.Add(ctx, account.CreateRequest{Phone: "+1"})
	_, err := svc.Add(ctx, account.CreateRequest{Phone: "+1"})
	if err == nil {
		t.Fatal("expected unique constraint error from postgres")
	}
}
