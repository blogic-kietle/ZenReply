package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedis initializes and returns a Redis client.
func NewRedis(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return rdb, nil
}
