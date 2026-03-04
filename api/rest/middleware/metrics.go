package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/metrics"
)

// Metrics returns a Gin middleware that records request count and latency.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath() // route pattern, e.g. /api/v1/accounts/:id
		if path == "" {
			path = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())
		elapsed := time.Since(start).Seconds()

		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(elapsed)
	}
}
