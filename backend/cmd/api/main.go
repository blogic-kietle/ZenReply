// ZenReply Backend API
//
//	@title						ZenReply API
//	@version					1.0.0
//	@description				ZenReply is an intelligent Slack auto-reply system for deep work sessions.
//	@termsOfService				https://zenreply.app/terms
//
//	@contact.name				ZenReply Support
//	@contact.url				https://zenreply.app/support
//	@contact.email				support@zenreply.app
//
//	@license.name				MIT
//	@license.url				https://opensource.org/licenses/MIT
//
//	@host						localhost:8080
//	@BasePath					/api/v1
//	@schemes					http https
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter: Bearer <your-jwt-token>
//
//	@tag.name					system
//	@tag.description			System health and diagnostics
//	@tag.name					auth
//	@tag.description			Slack OAuth 2.0 authentication flow
//	@tag.name					users
//	@tag.description			User profile management
//	@tag.name					deep-work
//	@tag.description			Deep work session management
//	@tag.name					settings
//	@tag.description			User auto-reply configuration
//	@tag.name					logs
//	@tag.description			Auto-reply message history
//	@tag.name					slack
//	@tag.description			Slack Events API webhook

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/kietle/zenreply/docs"
	"github.com/kietle/zenreply/config"
	"github.com/kietle/zenreply/handler"
	"github.com/kietle/zenreply/pkg/database"
	"github.com/kietle/zenreply/pkg/logger"
	slackpkg "github.com/kietle/zenreply/pkg/slack"
	"github.com/kietle/zenreply/repository"
	"github.com/kietle/zenreply/route"
	"github.com/kietle/zenreply/service"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.App.LogLevel)

	slog.SetDefault(log)

	log.Info("starting ZenReply API",
		slog.String("version", cfg.App.Version),
		slog.String("env", cfg.App.Env),
		slog.String("port", cfg.App.Port),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Database ─────────────────────────────────────────────────────────────
	db, err := database.NewPostgres(ctx, &cfg.Postgres)
	if err != nil {
		log.Error("failed to connect to PostgreSQL", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to PostgreSQL")

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb, err := database.NewRedis(ctx, &cfg.Redis)
	if err != nil {
		log.Error("failed to connect to Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to Redis")

	// ── Migrations ────────────────────────────────────────────────────────────
	if err := database.RunMigrations(ctx, db, log); err != nil {
		log.Error("failed to run database migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("database migrations completed")

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	messageLogRepo := repository.NewMessageLogRepository(db)

	// ── Slack Clients ─────────────────────────────────────────────────────────
	oauthSvc := slackpkg.NewOAuthService(&cfg.Slack)
	messenger := slackpkg.NewMessenger(log)

	// ── Services ──────────────────────────────────────────────────────────────
	authService := service.NewAuthService(cfg, userRepo, settingsRepo, oauthSvc, rdb)
	deepWorkService := service.NewDeepWorkService(sessionRepo, settingsRepo, userRepo, messageLogRepo, rdb, messenger, log)
	settingsService := service.NewSettingsService(settingsRepo)

	// ── HTTP Handler ──────────────────────────────────────────────────────────
	h := handler.New(cfg, db, rdb, authService, deepWorkService, settingsService)

	// ── Router ────────────────────────────────────────────────────────────────
	router := route.Setup(cfg, h)

	// ── Socket Mode (optional) ────────────────────────────────────────────────
	if cfg.Slack.AppToken != "" && cfg.Slack.BotToken != "" {
		socketClient := slackpkg.NewSocketModeClient(
			cfg.Slack.AppToken,
			cfg.Slack.BotToken,
			func(ctx context.Context, ev interface{}) {
				// Event handling is done inside the socket mode client.
			},
			log,
		)
		go func() {
			log.Info("starting Slack Socket Mode client")
			if err := socketClient.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Error("socket mode client error", slog.String("error", err.Error()))
			}
		}()
	} else {
		log.Info("slack socket mode disabled (SLACK_APP_TOKEN or SLACK_BOT_TOKEN not set)")
	}

	// ── HTTP Server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		log.Info("HTTP server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server gracefully...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	log.Info("server exited")
}
