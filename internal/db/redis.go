package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/bohemiyan/RBAC/internal/config"
)

// NewRedisClient initializes and returns a Redis client.
func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}
