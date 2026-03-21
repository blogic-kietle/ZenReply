// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/pkg/middleware"
	"github.com/kietle/zenreply/pkg/response"
)

// ListMessageLogs godoc
//
//	@Summary		List auto-reply message logs
//	@Description	Returns a paginated list of all auto-reply messages sent by ZenReply for the authenticated user
//	@Tags			logs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int	false	"Page number (default: 1)"
//	@Param			per_page	query	int	false	"Items per page (default: 20, max: 100)"
//	@Success		200	{object}	response.Response{data=[]model.MessageLog}
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/logs [get]
func (h *Handler) ListMessageLogs(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	logs, total, err := h.deepWorkService.ListMessageLogs(ctx, userID, perPage, offset)
	if err != nil {
		response.InternalServerError(c, "failed to retrieve message logs")
		return
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	response.OKWithMeta(c, "message logs retrieved", logs, &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// ListSessionMessageLogs godoc
//
//	@Summary		List message logs for a specific session
//	@Description	Returns all auto-reply messages sent during a specific deep work session
//	@Tags			logs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			session_id	path	string	true	"Session UUID"
//	@Success		200	{object}	response.Response{data=[]model.MessageLog}
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/logs/sessions/{session_id} [get]
func (h *Handler) ListSessionMessageLogs(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("session_id")

	logs, err := h.deepWorkService.ListSessionMessageLogs(ctx, sessionID)
	if err != nil {
		response.InternalServerError(c, "failed to retrieve session message logs")
		return
	}

	response.OK(c, "session message logs retrieved", logs)
}
