package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/api/rest/handler"
	"github.com/tgplane/tgplane/internal/webhook"
)

func newWebhookRouter() (*gin.Engine, *webhook.Service) {
	svc := webhook.NewService(webhook.NewMemoryRepository())
	r := gin.New()
	handler.NewWebhookHandler(svc).Register(r.Group("/api/v1"))
	return r, svc
}

func TestWebhookHandler_Create(t *testing.T) {
	r, _ := newWebhookRouter()

	body := `{"url":"https://example.com/hook","secret":"s3cr3t","events":["message"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookHandler_Create_InvalidURL(t *testing.T) {
	r, _ := newWebhookRouter()

	body := `{"url":"not-a-url"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWebhookHandler_List(t *testing.T) {
	r, svc := newWebhookRouter()
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	_, _ = svc.Create(ctx, "https://a.example.com/hook", "", nil)
	_, _ = svc.Create(ctx, "https://b.example.com/hook", "", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/webhooks", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 2 {
		t.Errorf("expected 2, got %d", len(list))
	}
}

func TestWebhookHandler_Delete(t *testing.T) {
	r, svc := newWebhookRouter()
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	wh, _ := svc.Create(ctx, "https://example.com/hook", "", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete,
		"/api/v1/webhooks/"+strconv.FormatInt(wh.ID, 10), nil))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestWebhookHandler_Delete_NotFound(t *testing.T) {
	r, _ := newWebhookRouter()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/webhooks/9999", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
