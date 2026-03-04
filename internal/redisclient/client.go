package redisclient

import (
	"github.com/redis/go-redis/v9"
	"github.com/tgplane/tgplane/internal/config"
)

func New(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}
