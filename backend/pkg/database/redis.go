// Package database provides database connection utilities.
package database

import (
	"context"
	"fmt"

	"github.com/kietle/zenreply/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis initializes and returns a Redis client using the provided config.
func NewRedis(ctx context.Context, cfg *config.RedisConfig) (*redis.Client, error) {
	var opts *redis.Options
	var err error

	if cfg.URL != "" {
		opts, err = redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL: %w", err)
		}
	} else {
		opts = &redis.Options{
			Addr:     cfg.Addr(),
			Password: cfg.Password,
			DB:       cfg.DB,
		}
	}

	rdb := redis.NewClient(opts)

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	// Enable keyspace notifications for expired events (used by deep work timer).
	// Requires Redis config: notify-keyspace-events "Ex"
	if err := rdb.ConfigSet(ctx, "notify-keyspace-events", "Ex").Err(); err != nil {
		// Non-fatal: some Redis deployments may not allow CONFIG SET.
		fmt.Printf("[redis] warning: could not set keyspace notifications: %v\n", err)
	}

	return rdb, nil
}
