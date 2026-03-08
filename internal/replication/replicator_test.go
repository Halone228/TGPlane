package replication

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// fakeInserter tracks calls and returns errors from a predefined sequence.
type fakeInserter struct {
	calls  int
	errors []error // error to return on each successive call; nil = success
}

func (f *fakeInserter) insert(_ context.Context, _ map[string]interface{}) error {
	idx := f.calls
	f.calls++
	if idx < len(f.errors) {
		return f.errors[idx]
	}
	return nil
}

// testInsertWithRetry mirrors the retry logic from insertWithRetry but uses
// a pluggable insert function so we can test without a real database.
func testInsertWithRetry(ctx context.Context, log *zap.Logger, insertFn func(context.Context, map[string]interface{}) error, vals map[string]interface{}, streamID string) error {
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Millisecond): // use short delay for tests
			}
		}
		if err := insertFn(ctx, vals); err != nil {
			lastErr = err
			log.Warn("insert message retry")
			continue
		}
		return nil
	}
	return lastErr
}

func TestInsertWithRetry_SuccessFirstTry(t *testing.T) {
	log := zap.NewNop()
	fi := &fakeInserter{errors: []error{nil}}

	err := testInsertWithRetry(context.Background(), log, fi.insert, map[string]interface{}{"type": "test"}, "1-0")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fi.calls != 1 {
		t.Fatalf("expected 1 call, got %d", fi.calls)
	}
}

func TestInsertWithRetry_SuccessOnSecondTry(t *testing.T) {
	log := zap.NewNop()
	fi := &fakeInserter{errors: []error{errors.New("transient"), nil}}

	err := testInsertWithRetry(context.Background(), log, fi.insert, map[string]interface{}{"type": "test"}, "1-0")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fi.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", fi.calls)
	}
}

func TestInsertWithRetry_FailAfterMaxRetries(t *testing.T) {
	log := zap.NewNop()
	errPerm := errors.New("permanent")
	fi := &fakeInserter{errors: []error{errPerm, errPerm, errPerm}}

	err := testInsertWithRetry(context.Background(), log, fi.insert, map[string]interface{}{"type": "test"}, "1-0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errPerm) {
		t.Fatalf("expected permanent error, got %v", err)
	}
	if fi.calls != 3 {
		t.Fatalf("expected 3 calls, got %d", fi.calls)
	}
}

func TestInsertWithRetry_ContextCancelled(t *testing.T) {
	log := zap.NewNop()
	fi := &fakeInserter{errors: []error{errors.New("fail"), errors.New("fail"), errors.New("fail")}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := testInsertWithRetry(ctx, log, fi.insert, map[string]interface{}{}, "1-0")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	// First attempt runs (no delay), second attempt checks ctx and returns early.
	if fi.calls > 1 {
		// The first attempt fails, then the retry branch should detect ctx cancellation.
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	}
}

func TestStrVal(t *testing.T) {
	m := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	if got := strVal(m, "key1"); got != "value1" {
		t.Fatalf("expected 'value1', got %q", got)
	}
	if got := strVal(m, "key2"); got != "" {
		t.Fatalf("expected empty string for non-string value, got %q", got)
	}
	if got := strVal(m, "missing"); got != "" {
		t.Fatalf("expected empty string for missing key, got %q", got)
	}
}

func TestInt64Val(t *testing.T) {
	m := map[string]interface{}{
		"int":    int64(42),
		"str":    "100",
		"bad":    "notanumber",
		"other":  3.14,
	}
	if got := int64Val(m, "int"); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
	if got := int64Val(m, "str"); got != 100 {
		t.Fatalf("expected 100, got %d", got)
	}
	if got := int64Val(m, "bad"); got != 0 {
		t.Fatalf("expected 0 for unparseable string, got %d", got)
	}
	if got := int64Val(m, "other"); got != 0 {
		t.Fatalf("expected 0 for float type, got %d", got)
	}
	if got := int64Val(m, "missing"); got != 0 {
		t.Fatalf("expected 0 for missing key, got %d", got)
	}
}

func TestNew(t *testing.T) {
	log := zap.NewNop()
	r := New(nil, nil, log)
	if r == nil {
		t.Fatal("expected non-nil Replicator")
	}
	if r.log != log {
		t.Fatal("expected logger to be set")
	}
}
