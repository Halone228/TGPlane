package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tgplane/tgplane/internal/config"
	"github.com/tgplane/tgplane/internal/logger"
	"github.com/tgplane/tgplane/internal/metrics"
	"github.com/tgplane/tgplane/internal/session"
	"github.com/tgplane/tgplane/internal/tdlib"
	workerserver "github.com/tgplane/tgplane/internal/worker/server"
	"go.uber.org/zap"
)

func main() {
	cfgPath := flag.String("config", "config.worker.yaml", "path to config file")
	workerID := flag.String("id", "", "unique worker ID (overrides config app.name)")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	l := logger.Must(cfg.Log.Level, cfg.Log.JSON)
	defer l.Sync() //nolint:errcheck

	id := cfg.App.Name
	if *workerID != "" {
		id = *workerID
	}

	metrics.BuildInfo.WithLabelValues("0.1.0", id).Set(1)

	pool := session.NewPool(
		func(sessID, phone, token string) (session.TDClient, error) {
			return tdlib.New(tdlib.SessionConfig{
				SessionID:   sessID,
				PhoneNumber: phone,
				BotToken:    token,
				APIID:       cfg.TDLib.APIID,
				APIHash:     cfg.TDLib.APIHash,
				DataDir:     cfg.TDLib.DataDir,
				LogLevel:    cfg.TDLib.LogLevel,
				UseTestDC:   cfg.TDLib.UseTestDC,
			}, l)
		},
		nil, // update handler is set by WorkerServer
		l,
		metrics.NewSessionHook(),
	)

	// --- gRPC worker server ---
	workerSrv := workerserver.New(id, pool, l)
	grpcSrv := workerserver.NewGRPCServer(workerSrv, l)

	go func() {
		if err := grpcSrv.Serve(cfg.GRPC.ListenAddr); err != nil {
			l.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	l.Info("TGPlane worker node started",
		zap.String("id", id),
		zap.String("grpc", cfg.GRPC.ListenAddr),
		zap.String("main", cfg.GRPC.MainAddr),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("shutting down…")
	grpcSrv.Stop()
	l.Info("stopped")
}
