// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/pkg/response"
)

// HealthResponse is the response body for the health check endpoint.
type HealthResponse struct {
	Status    string `json:"status" example:"ok"`
	Service   string `json:"service" example:"ZenReply API"`
	Version   string `json:"version" example:"1.0.0"`
	Timestamp string `json:"timestamp" example:"2024-01-01T00:00:00Z"`
	Database  string `json:"database" example:"ok"`
	Redis     string `json:"redis" example:"ok"`
}

// HealthCheck godoc
//
//	@Summary		Health check
//	@Description	Returns the health status of the API and its dependencies
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	response.Response{data=HealthResponse}
//	@Failure		503	{object}	response.Response
//	@Router			/health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	dbStatus := "ok"
	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "error: " + err.Error()
	}

	redisStatus := "ok"
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		redisStatus = "error: " + err.Error()
	}

	data := HealthResponse{
		Status:    "ok",
		Service:   h.cfg.App.Name,
		Version:   h.cfg.App.Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
		Redis:     redisStatus,
	}

	if dbStatus != "ok" || redisStatus != "ok" {
		c.JSON(http.StatusServiceUnavailable, response.Response{
			Success: false,
			Data:    data,
		})
		return
	}

	response.OK(c, "service is healthy", data)
}
