package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/account"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newAccountRouter() *gin.Engine {
	svc := account.NewService(account.NewMemoryRepository(), zap.NewNop())
	r := gin.New()
	NewAccountHandler(svc).Register(r.Group("/api/v1"))
	return r
}

func TestAccountHandler_Create(t *testing.T) {
	r := newAccountRouter()

	body, _ := json.Marshal(map[string]string{"phone": "+79001234567"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp account.Account
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Phone != "+79001234567" {
		t.Errorf("unexpected phone: %s", resp.Phone)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestAccountHandler_Create_InvalidBody(t *testing.T) {
	r := newAccountRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAccountHandler_Create_MissingPhone(t *testing.T) {
	r := newAccountRouter()

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing phone, got %d", w.Code)
	}
}

func TestAccountHandler_List(t *testing.T) {
	r := newAccountRouter()

	// Create two accounts first.
	for _, phone := range []string{"+1", "+2"} {
		body, _ := json.Marshal(map[string]string{"phone": phone})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var accounts []account.Account
	if err := json.NewDecoder(w.Body).Decode(&accounts); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestAccountHandler_Get(t *testing.T) {
	r := newAccountRouter()

	// Create account.
	body, _ := json.Marshal(map[string]string{"phone": "+1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created account.Account
	json.NewDecoder(w.Body).Decode(&created)

	// Get by ID.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/accounts/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got account.Account
	json.NewDecoder(w.Body).Decode(&got)
	if got.ID != created.ID {
		t.Errorf("ID mismatch: want %d, got %d", created.ID, got.ID)
	}
}

func TestAccountHandler_Get_NotFound(t *testing.T) {
	r := newAccountRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAccountHandler_Delete(t *testing.T) {
	r := newAccountRouter()

	body, _ := json.Marshal(map[string]string{"phone": "+1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Verify deleted.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/accounts/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestAccountHandler_Get_InvalidID(t *testing.T) {
	r := newAccountRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric ID, got %d", w.Code)
	}
}
