package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgplanev1 "github.com/tgplane/tgplane/api/proto/gen/tgplane/v1"
	"github.com/tgplane/tgplane/api/rest"
	"github.com/tgplane/tgplane/internal/account"
	"github.com/tgplane/tgplane/internal/auth"
	"github.com/tgplane/tgplane/internal/bot"
	"github.com/tgplane/tgplane/internal/bulk"
	"github.com/tgplane/tgplane/internal/config"
	"github.com/tgplane/tgplane/internal/database"
	"github.com/tgplane/tgplane/internal/logger"
	"github.com/tgplane/tgplane/internal/metrics"
	"github.com/tgplane/tgplane/internal/redisclient"
	"github.com/tgplane/tgplane/internal/replication"
	"github.com/tgplane/tgplane/internal/session"
	"github.com/tgplane/tgplane/internal/stream"
	"github.com/tgplane/tgplane/internal/webhook"
	"github.com/tgplane/tgplane/internal/worker/manager"
	"go.uber.org/zap"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	l := logger.Must(cfg.Log.Level, cfg.Log.JSON)
	defer l.Sync() //nolint:errcheck

	// --- Database ---
	db, err := database.Connect(cfg.Database)
	if err != nil {
		l.Fatal("connect to postgres", zap.Error(err))
	}
	defer db.Close()
	l.Info("connected to postgres")

	if err := database.Migrate(db, "migrations"); err != nil {
		l.Fatal("run migrations", zap.Error(err))
	}
	l.Info("migrations applied")

	// --- Redis ---
	rdb := redisclient.New(cfg.Redis)
	defer rdb.Close()

	// --- Repositories & services ---
	accountRepo := account.NewPostgresRepository(db)
	botRepo := bot.NewPostgresRepository(db)
	authRepo := auth.NewPostgresRepository(db)
	webhookRepo := webhook.NewPostgresRepository(db)

	accountSvc := account.NewService(accountRepo, l)
	botSvc := bot.NewService(botRepo, l)
	authSvc := auth.NewService(authRepo, cfg.Auth.MasterKey)
	webhookSvc := webhook.NewService(webhookRepo)

	// --- Stream publisher ---
	publisher := stream.NewPublisher(rdb)

	// --- Session pool ---
	// ClientFactory will be wired to tdlib once it's available.
	// For now it returns an error — sessions are managed by worker nodes.
	pool := session.NewPool(
		func(id, phone, token string) (session.TDClient, error) {
			return nil, errors.New("tdlib not available on main node; use a worker")
		},
		func(sessionID string, update interface{}) {
			l.Debug("update received", zap.String("session", sessionID))
		},
		l,
		metrics.NewSessionHook(),
	)

	metrics.BuildInfo.WithLabelValues("0.1.0", "main").Set(1)

	// --- Worker manager ---
	workerMgr := manager.New(func(workerID string, upd *tgplanev1.TelegramUpdate) {
		l.Debug("update routed",
			zap.String("worker", workerID),
			zap.String("session", upd.SessionId),
			zap.String("type", upd.Type),
		)
		if err := publisher.Publish(context.Background(), workerID, upd); err != nil {
			l.Error("publish update to stream", zap.Error(err))
		}
	}, l)

	// --- Bulk service ---
	bulkSvc := bulk.NewService(accountSvc, botSvc, workerMgr)

	// --- Webhook dispatcher ---
	dispatcher := webhook.NewDispatcher(rdb, webhookRepo, l)

	// --- Message replicator ---
	replicator := replication.New(rdb, db, l)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- HTTP server ---
	srv := rest.NewServer(rest.Deps{
		AccountSvc: accountSvc,
		BotSvc:     botSvc,
		Pool:       pool,
		WorkerMgr:  workerMgr,
		AuthSvc:    authSvc,
		WebhookSvc: webhookSvc,
		BulkSvc:    bulkSvc,
		RateLimit:  cfg.RateLimit,
		AppCtx:     ctx,
		Log:        l,
		Addr:       cfg.HTTP.Addr,
	})

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Fatal("HTTP server error", zap.Error(err))
		}
	}()
	go dispatcher.Run(ctx)
	go replicator.Run(ctx)

	l.Info("TGPlane main node started",
		zap.String("http", cfg.HTTP.Addr),
		zap.String("grpc", cfg.GRPC.ListenAddr),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("shutting down…")
	cancel() // stop dispatcher

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		l.Error("HTTP shutdown error", zap.Error(err))
	}
	l.Info("stopped")
}
