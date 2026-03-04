package session

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func benchPool(b *testing.B, n int) *Pool {
	b.Helper()
	return NewPool(
		func(id, _, _ string) (TDClient, error) { return newMockClient(id), nil },
		noopHandler,
		zap.NewNop(),
		nil,
	)
}

// BenchmarkPool_Add measures Add+Remove cycle to avoid unbounded goroutine growth.
func BenchmarkPool_Add(b *testing.B) {
	ctx := context.Background()
	p := benchPool(b, 0)
	const id = "bench-sess"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := p.Add(ctx, id, "+1"); err != nil {
			b.Fatal(err)
		}
		if err := p.Remove(id); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPool_Get measures concurrent read throughput (hot path: every API call hits this).
func BenchmarkPool_Get(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := benchPool(b, 0)
	const sessions = 1000
	for i := 0; i < sessions; i++ {
		_ = p.Add(ctx, fmt.Sprintf("sess-%d", i), "+1")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			p.Get(fmt.Sprintf("sess-%d", i%sessions))
			i++
		}
	})
}

// BenchmarkPool_AddRemove measures the Add+Remove cycle under parallel load.
func BenchmarkPool_AddRemove(b *testing.B) {
	ctx := context.Background()
	p := benchPool(b, 0)

	var mu sync.Mutex
	counter := 0

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			id := fmt.Sprintf("sess-%d", counter)
			counter++
			mu.Unlock()

			_ = p.Add(ctx, id, "+1")
			_ = p.Remove(id)
		}
	})
}

// BenchmarkPool_List measures snapshot cost at 1000 sessions.
func BenchmarkPool_List(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := benchPool(b, 0)
	for i := 0; i < 1000; i++ {
		_ = p.Add(ctx, fmt.Sprintf("sess-%d", i), "+1")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.List()
	}
}
