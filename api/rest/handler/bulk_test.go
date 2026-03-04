package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/tgplane/tgplane/api/rest/handler"
	"github.com/tgplane/tgplane/internal/account"
	"github.com/tgplane/tgplane/internal/bot"
	"github.com/tgplane/tgplane/internal/bulk"
)

func newBulkRouter() *gin.Engine {
	accountSvc := account.NewService(account.NewMemoryRepository(), zap.NewNop())
	botSvc := bot.NewService(bot.NewMemoryRepository(), zap.NewNop())
	svc := bulk.NewService(accountSvc, botSvc, nil)
	r := gin.New()
	handler.NewBulkHandler(svc).Register(r.Group("/api/v1"))
	return r
}

func TestBulkHandler_AddBots(t *testing.T) {
	r := newBulkRouter()

	body := `{"items":[{"token":"1:AAA"},{"token":"2:BBB"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bulk/bots", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", w.Code, w.Body.String())
	}
	var result bulk.BulkResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 2 || result.Succeeded != 2 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestBulkHandler_AddBots_EmptyItems(t *testing.T) {
	r := newBulkRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bulk/bots",
		bytes.NewBufferString(`{"items":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBulkHandler_AddAccounts(t *testing.T) {
	r := newBulkRouter()

	body := `{"items":[{"phone":"+79001234567"},{"phone":"+79007654321"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bulk/accounts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBulkHandler_RemoveSessions(t *testing.T) {
	r := newBulkRouter()

	body := `{"session_ids":["nonexistent-1","nonexistent-2"]}`
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bulk/sessions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", w.Code, w.Body.String())
	}
	var result bulk.BulkResult
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if result.Failed != 2 {
		t.Errorf("expected 2 failures for nonexistent sessions, got %d", result.Failed)
	}
}
