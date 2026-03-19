package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgres creates a new PostgreSQL connection pool.
func NewPostgres(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	pgx, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configuration
	pgx.PingTimeout = 15 * time.Second
	pgx.MaxConns = 25
	pgx.MinConns = 2
	pgx.MaxConnLifetime = 5 * time.Minute
	pgx.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pgx)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
