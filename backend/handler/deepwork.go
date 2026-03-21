// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/pkg/middleware"
	"github.com/kietle/zenreply/pkg/response"
)

// StartSessionRequest is the request body for starting a deep work session.
type StartSessionRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"Focused coding sprint"`
}

// StartSession godoc
//
//	@Summary		Start a deep work session
//	@Description	Begins a new deep work session for the authenticated user. Any existing active session is ended automatically.
//	@Tags			deep-work
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		StartSessionRequest	true	"Session details"
//	@Success		201		{object}	response.Response{data=model.DeepWorkSession}
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		500		{object}	response.Response
//	@Router			/deep-work/sessions [post]
func (h *Handler) StartSession(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "invalid request body", err.Error())
		return
	}

	session, err := h.deepWorkService.StartSession(ctx, userID, req.Reason)
	if err != nil {
		response.InternalServerError(c, "failed to start deep work session")
		return
	}

	response.Created(c, "deep work session started", session)
}

// EndSession godoc
//
//	@Summary		End the active deep work session
//	@Description	Terminates the currently active deep work session for the authenticated user
//	@Tags			deep-work
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=model.DeepWorkSession}
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/deep-work/sessions/active [delete]
func (h *Handler) EndSession(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	session, err := h.deepWorkService.EndSession(ctx, userID)
	if err != nil {
		if err.Error() == "no active deep work session found" {
			response.NotFound(c, "no active deep work session found")
			return
		}
		response.InternalServerError(c, "failed to end deep work session")
		return
	}

	response.OK(c, "deep work session ended", session)
}

// GetStatus godoc
//
//	@Summary		Get current deep work status
//	@Description	Returns whether the authenticated user is currently in a deep work session
//	@Tags			deep-work
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=model.DeepWorkStatus}
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/deep-work/status [get]
func (h *Handler) GetStatus(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	status, err := h.deepWorkService.GetStatus(ctx, userID)
	if err != nil {
		response.InternalServerError(c, "failed to retrieve deep work status")
		return
	}

	response.OK(c, "deep work status retrieved", status)
}

// ListSessions godoc
//
//	@Summary		List deep work sessions
//	@Description	Returns a paginated list of all deep work sessions for the authenticated user
//	@Tags			deep-work
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int	false	"Page number (default: 1)"
//	@Param			per_page	query	int	false	"Items per page (default: 20, max: 100)"
//	@Success		200	{object}	response.Response{data=[]model.DeepWorkSession}
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/deep-work/sessions [get]
func (h *Handler) ListSessions(c *gin.Context) {
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
	sessions, total, err := h.deepWorkService.ListSessions(ctx, userID, perPage, offset)
	if err != nil {
		response.InternalServerError(c, "failed to list sessions")
		return
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	response.OKWithMeta(c, "sessions retrieved", sessions, &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetSession godoc
//
//	@Summary		Get a specific session by ID
//	@Description	Returns details of a specific deep work session
//	@Tags			deep-work
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Session UUID"
//	@Success		200	{object}	response.Response{data=model.DeepWorkSession}
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/deep-work/sessions/{id} [get]
func (h *Handler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("id")

	session, err := h.deepWorkService.GetSessionByID(ctx, sessionID)
	if err != nil {
		response.NotFound(c, "session not found")
		return
	}

	response.OK(c, "session retrieved", session)
}
