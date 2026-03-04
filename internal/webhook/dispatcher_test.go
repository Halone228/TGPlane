package webhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tgplane/tgplane/internal/webhook"
)

func TestDispatcher_HTTPDelivery(t *testing.T) {
	received := make(chan map[string]interface{}, 1)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var ev map[string]interface{}
		_ = json.Unmarshal(body, &ev)
		received <- ev
	}))
	defer target.Close()

	repo := webhook.NewMemoryRepository()
	_, _ = repo.Create(context.Background(), target.URL, "s3cr3t", []string{})

	d := webhook.NewDispatcher(nil, repo, zap.NewNop())

	ev := webhook.UpdateEvent{SessionID: "s1", WorkerID: "w1", Type: "message"}
	body, _ := json.Marshal(ev)
	d.DeliverBody(context.Background(), body, ev.Type)

	select {
	case got := <-received:
		if got["session_id"] != "s1" {
			t.Errorf("wrong session_id: %v", got["session_id"])
		}
	case <-time.After(3 * time.Second):
		t.Error("timed out waiting for webhook delivery")
	}
}

func TestDispatcher_EventFilter_Skips(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer target.Close()

	repo := webhook.NewMemoryRepository()
	// webhook only accepts "photo" events
	_, _ = repo.Create(context.Background(), target.URL, "", []string{"photo"})

	d := webhook.NewDispatcher(nil, repo, zap.NewNop())

	ev := webhook.UpdateEvent{Type: "message"}
	body, _ := json.Marshal(ev)
	d.DeliverBody(context.Background(), body, ev.Type)

	time.Sleep(100 * time.Millisecond)
	if called {
		t.Error("webhook should have been filtered out")
	}
}
