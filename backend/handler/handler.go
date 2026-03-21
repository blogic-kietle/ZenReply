// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kietle/zenreply/config"
	"github.com/kietle/zenreply/service"
	"github.com/redis/go-redis/v9"
)

// Handler holds all service dependencies required by the HTTP handlers.
type Handler struct {
	cfg             *config.Config
	db              *pgxpool.Pool
	rdb             *redis.Client
	authService     *service.AuthService
	deepWorkService *service.DeepWorkService
	settingsService *service.SettingsService
}

// New creates a new Handler with all required dependencies.
func New(
	cfg *config.Config,
	db *pgxpool.Pool,
	rdb *redis.Client,
	authService *service.AuthService,
	deepWorkService *service.DeepWorkService,
	settingsService *service.SettingsService,
) *Handler {
	return &Handler{
		cfg:             cfg,
		db:              db,
		rdb:             rdb,
		authService:     authService,
		deepWorkService: deepWorkService,
		settingsService: settingsService,
	}
}
