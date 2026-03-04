package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health registers the /health and /ready endpoints.
func Health(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/ready", func(c *gin.Context) {
		// TODO: check DB and Redis connectivity
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
