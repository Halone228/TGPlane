package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/api/rest/middleware"
	"github.com/tgplane/tgplane/internal/auth"
)

func init() { gin.SetMode(gin.TestMode) }

func newRouter(svc *auth.Service) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Auth(svc))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	svc := auth.NewService(auth.NewMemoryRepository(), "")
	r := newRouter(svc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	svc := auth.NewService(auth.NewMemoryRepository(), "")
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Api-Key", "badkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidKey(t *testing.T) {
	svc := auth.NewService(auth.NewMemoryRepository(), "")
	_, raw, _ := svc.Create(context.Background(), "test")
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Api-Key", raw)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_MasterKey(t *testing.T) {
	svc := auth.NewService(auth.NewMemoryRepository(), "masterkey")
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Api-Key", "masterkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
