package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/bot"
	"go.uber.org/zap"
)

func newBotRouter() *gin.Engine {
	svc := bot.NewService(bot.NewMemoryRepository(), zap.NewNop())
	r := gin.New()
	NewBotHandler(svc).Register(r.Group("/api/v1"))
	return r
}

func TestBotHandler_Create(t *testing.T) {
	r := newBotRouter()

	body, _ := json.Marshal(map[string]string{"token": "123:ABC"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp bot.Bot
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Token != "123:ABC" {
		t.Errorf("unexpected token: %s", resp.Token)
	}
}

func TestBotHandler_Create_MissingToken(t *testing.T) {
	r := newBotRouter()

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBotHandler_List(t *testing.T) {
	r := newBotRouter()

	for _, tok := range []string{"tok1", "tok2"} {
		body, _ := json.Marshal(map[string]string{"token": tok})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/bots", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var bots []bot.Bot
	json.NewDecoder(w.Body).Decode(&bots)
	if len(bots) != 2 {
		t.Errorf("expected 2 bots, got %d", len(bots))
	}
}

func TestBotHandler_Delete(t *testing.T) {
	r := newBotRouter()

	body, _ := json.Marshal(map[string]string{"token": "tok"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/bots/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestBotHandler_Get_NotFound(t *testing.T) {
	r := newBotRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bots/9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
