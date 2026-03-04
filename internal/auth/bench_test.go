package auth

import (
	"context"
	"testing"
)

// BenchmarkHashKey measures SHA-256 hashing — called on every API request.
func BenchmarkHashKey(b *testing.B) {
	key := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f60000"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hashKey(key)
	}
}

// BenchmarkGenerateKey measures crypto/rand key generation.
func BenchmarkGenerateKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := generateKey(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidate_MemoryRepo measures full Validate path (hash + map lookup).
func BenchmarkValidate_MemoryRepo(b *testing.B) {
	svc := NewService(NewMemoryRepository(), "")
	ctx := context.Background()

	_, raw, err := svc.Create(ctx, "bench-key")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Validate(ctx, raw)
	}
}

// BenchmarkValidate_MasterKey measures the fast path (master key bypass).
func BenchmarkValidate_MasterKey(b *testing.B) {
	svc := NewService(NewMemoryRepository(), "supersecret")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Validate(ctx, "supersecret")
	}
}

// BenchmarkValidate_Parallel measures Validate under concurrent load.
func BenchmarkValidate_Parallel(b *testing.B) {
	svc := NewService(NewMemoryRepository(), "")
	ctx := context.Background()

	_, raw, err := svc.Create(ctx, "bench-key")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			svc.Validate(ctx, raw)
		}
	})
}
