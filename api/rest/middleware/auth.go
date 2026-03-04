package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tgplane/tgplane/internal/auth"
)

// Auth returns a middleware that validates X-Api-Key against the auth service.
func Auth(svc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Api-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-Api-Key header required"})
			return
		}
		if !svc.Validate(c.Request.Context(), key) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}
		c.Next()
	}
}
