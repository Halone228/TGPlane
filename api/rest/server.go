package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tgplane/tgplane/api/rest/handler"
	"github.com/tgplane/tgplane/api/rest/middleware"
	"github.com/tgplane/tgplane/internal/account"
	"github.com/tgplane/tgplane/internal/auth"
	"github.com/tgplane/tgplane/internal/bot"
	"github.com/tgplane/tgplane/internal/bulk"
	"github.com/tgplane/tgplane/internal/config"
	"github.com/tgplane/tgplane/internal/session"
	"github.com/tgplane/tgplane/internal/webhook"
	"github.com/tgplane/tgplane/internal/worker/manager"
	"go.uber.org/zap"
)

type Server struct {
	http *http.Server
	log  *zap.Logger
}

type Deps struct {
	AccountSvc *account.Service
	BotSvc     *bot.Service
	Pool       *session.Pool
	WorkerMgr  *manager.Manager
	AuthSvc    *auth.Service
	WebhookSvc *webhook.Service
	BulkSvc    *bulk.Service
	RateLimit  config.RateLimitConfig
	AppCtx     context.Context // for worker subscribe loops started via API
	Log        *zap.Logger
	Addr       string
}

func NewServer(d Deps) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.Recovery(d.Log))
	r.Use(middleware.Logger(d.Log))
	r.Use(middleware.Metrics())
	if d.RateLimit.RPS > 0 {
		r.Use(middleware.KeyRateLimiter(d.RateLimit.RPS, d.RateLimit.Burst))
	}

	handler.Health(r)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Serve Web UI from web/dist if present.
	r.Static("/ui", "./web/dist")
	r.GET("/", func(c *gin.Context) { c.Redirect(302, "/ui") })

	v1 := r.Group("/api/v1")

	// Auth key management (unprotected — needed to bootstrap first key)
	if d.AuthSvc != nil {
		handler.NewAuthHandler(d.AuthSvc).Register(v1)
	}

	// All other endpoints require a valid API key.
	if d.AuthSvc != nil {
		v1.Use(middleware.Auth(d.AuthSvc))
	}

	handler.NewAccountHandler(d.AccountSvc).Register(v1)
	handler.NewBotHandler(d.BotSvc).Register(v1)
	handler.NewSessionHandler(d.Pool).Register(v1)
	if d.WorkerMgr != nil {
		appCtx := d.AppCtx
		if appCtx == nil {
			appCtx = context.Background()
		}
		handler.NewWorkerHandler(d.WorkerMgr, appCtx).Register(v1)
	}
	if d.WebhookSvc != nil {
		handler.NewWebhookHandler(d.WebhookSvc).Register(v1)
	}
	if d.BulkSvc != nil {
		handler.NewBulkHandler(d.BulkSvc).Register(v1)
	}

	return &Server{
		http: &http.Server{
			Addr:         d.Addr,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		log: d.Log,
	}
}

func (s *Server) Start() error {
	s.log.Info("HTTP server listening", zap.String("addr", s.http.Addr))
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
