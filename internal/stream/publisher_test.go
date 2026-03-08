package stream

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestNewPublisher(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	pub := NewPublisher(rdb)
	if pub == nil {
		t.Fatal("expected non-nil Publisher")
	}
}

func TestNewPublisher_NilClient(t *testing.T) {
	pub := NewPublisher(nil)
	if pub == nil {
		t.Fatal("expected non-nil Publisher even with nil client")
	}
}

func TestUpdatesStreamConstant(t *testing.T) {
	if UpdatesStream != "tgplane:updates" {
		t.Fatalf("UpdatesStream = %q, want %q", UpdatesStream, "tgplane:updates")
	}
}
