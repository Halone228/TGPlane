package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type HealthDeps struct {
	DB  *sqlx.DB
	RDB *redis.Client
}

func Health(r *gin.Engine, deps HealthDeps) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if deps.DB != nil {
			if err := deps.DB.PingContext(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "error",
					"detail": "postgres: " + err.Error(),
				})
				return
			}
		}
		if deps.RDB != nil {
			if err := deps.RDB.Ping(ctx).Err(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "error",
					"detail": "redis: " + err.Error(),
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
