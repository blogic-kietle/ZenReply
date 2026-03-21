// Package route configures all HTTP routes for the ZenReply API.
package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/config"
	"github.com/kietle/zenreply/handler"
	"github.com/kietle/zenreply/pkg/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Setup configures and returns the main Gin router with all routes registered.
func Setup(cfg *config.Config, h *handler.Handler) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware.
	r.Use(middleware.Recovery(nil))
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS([]string{
		cfg.App.FrontendURL,
		"http://localhost:4200",
		"http://localhost:3000",
	}))

	// Swagger UI (Swaggo).
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Scalar OpenAPI UI — served as a self-contained HTML page.
	r.GET("/scalar", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, scalarHTML(cfg.App.BaseURL))
	})

	// Redirect root to Scalar docs.
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/scalar")
	})

	// OpenAPI JSON spec endpoint.
	r.GET("/openapi.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})

	// ── Health ──────────────────────────────────────────────────────────────
	r.GET("/health", h.HealthCheck)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// ── API v1 ───────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")

	// Auth (public)
	auth := v1.Group("/slack")
	{
		auth.GET("/auth", h.SlackAuthURL)
		auth.GET("/callback", h.SlackCallback)
		auth.GET("/callback/redirect", h.SlackCallbackRedirect)
		auth.POST("/events", middleware.SlackVerify(cfg.Slack.SigningSecret), h.SlackEventsWebhook)
	}

	// Protected routes (require JWT)
	protected := v1.Group("")
	protected.Use(middleware.Auth(cfg.JWT.Secret))
	{
		// Users
		users := protected.Group("/users")
		{
			users.GET("/me", h.GetMe)
			users.DELETE("/me", h.DeleteMe)
		}

		// Deep Work Sessions
		dw := protected.Group("/deep-work")
		{
			dw.GET("/status", h.GetStatus)
			dw.POST("/sessions", h.StartSession)
			dw.DELETE("/sessions/active", h.EndSession)
			dw.GET("/sessions", h.ListSessions)
			dw.GET("/sessions/:id", h.GetSession)
		}

		// Settings
		settings := protected.Group("/settings")
		{
			settings.GET("", h.GetSettings)
			settings.PUT("", h.UpdateSettings)
			settings.POST("/reset", h.ResetSettings)
			settings.GET("/whitelist", h.GetWhitelist)
			settings.POST("/whitelist", h.AddToWhitelist)
			settings.DELETE("/whitelist/:slack_user_id", h.RemoveFromWhitelist)
			settings.GET("/blacklist", h.GetBlacklist)
			settings.POST("/blacklist", h.AddToBlacklist)
			settings.DELETE("/blacklist/:slack_user_id", h.RemoveFromBlacklist)
		}

		// Message Logs
		logs := protected.Group("/logs")
		{
			logs.GET("", h.ListMessageLogs)
			logs.GET("/sessions/:session_id", h.ListSessionMessageLogs)
		}
	}

	// 404 handler.
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "the requested endpoint does not exist",
			},
		})
	})

	return r
}

// scalarHTML returns the Scalar API Reference HTML page.
func scalarHTML(baseURL string) string {
	return `<!doctype html>
<html>
  <head>
    <title>ZenReply API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      body { margin: 0; padding: 0; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="` + baseURL + `/openapi.json"
      data-configuration='{"theme":"purple","layout":"modern","defaultHttpClient":{"targetKey":"javascript","clientKey":"fetch"},"hideDownloadButton":false}'
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`
}
