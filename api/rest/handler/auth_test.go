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
	"github.com/tgplane/tgplane/internal/auth"
)

func init() { gin.SetMode(gin.TestMode) }

func newAuthRouter() (*gin.Engine, *auth.Service) {
	svc := auth.NewService(auth.NewMemoryRepository(), "")
	r := gin.New()
	handler.NewAuthHandler(svc).Register(r.Group("/api/v1"))
	return r, svc
}

func TestAuthHandler_Create(t *testing.T) {
	r, _ := newAuthRouter()

	body := `{"name":"test-key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["key"] == nil {
		t.Error("expected raw key in response")
	}
}

func TestAuthHandler_Create_MissingName(t *testing.T) {
	r, _ := newAuthRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/keys", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_List(t *testing.T) {
	r, svc := newAuthRouter()
	ctx := req().Context()
	_, _, _ = svc.Create(ctx, "k1")
	_, _, _ = svc.Create(ctx, "k2")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/keys", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 keys, got %d", len(list))
	}
}

func TestAuthHandler_Delete(t *testing.T) {
	r, svc := newAuthRouter()
	ctx := req().Context()
	k, _, _ := svc.Create(ctx, "temp")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/auth/keys/"+strconv.FormatInt(k.ID, 10), nil))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Delete_NotFound(t *testing.T) {
	r, _ := newAuthRouter()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/auth/keys/9999", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func req() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/", nil)
}
