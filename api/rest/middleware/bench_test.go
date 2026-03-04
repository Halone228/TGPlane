package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/api/rest/middleware"
	"github.com/tgplane/tgplane/internal/auth"
)

func newBenchRouter(mw ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	for _, m := range mw {
		r.Use(m)
	}
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

// BenchmarkAuth_ValidKey measures auth middleware with a valid key (hash lookup).
func BenchmarkAuth_ValidKey(b *testing.B) {
	svc := auth.NewService(auth.NewMemoryRepository(), "")
	ctx := b.Context()
	_, raw, err := svc.Create(ctx, "bench")
	if err != nil {
		b.Fatal(err)
	}

	r := newBenchRouter(middleware.Auth(svc))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.Header.Set("X-Api-Key", raw)
			r.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}

// BenchmarkAuth_MasterKey measures the fast path (master key bypass, no hash).
func BenchmarkAuth_MasterKey(b *testing.B) {
	svc := auth.NewService(auth.NewMemoryRepository(), "masterkey")
	r := newBenchRouter(middleware.Auth(svc))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.Header.Set("X-Api-Key", "masterkey")
			r.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}

// BenchmarkKeyRateLimiter_SameKey measures per-key limiter with a single key (map hit).
func BenchmarkKeyRateLimiter_SameKey(b *testing.B) {
	r := newBenchRouter(middleware.KeyRateLimiter(1e9, 1e9))
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Api-Key", "fixed-key")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}

// BenchmarkKeyRateLimiter_ManyKeys measures limiter with N distinct keys (map growth).
func BenchmarkKeyRateLimiter_ManyKeys(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("keys=%d", n), func(b *testing.B) {
			r := newBenchRouter(middleware.KeyRateLimiter(1e9, 1e9))
			reqs := make([]*http.Request, n)
			for i := 0; i < n; i++ {
				req := httptest.NewRequest(http.MethodGet, "/ping", nil)
				req.Header.Set("X-Api-Key", fmt.Sprintf("key-%d", i))
				reqs[i] = req
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					r.ServeHTTP(httptest.NewRecorder(), reqs[i%n])
					i++
				}
			})
		})
	}
}
