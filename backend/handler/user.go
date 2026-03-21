// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/pkg/middleware"
	"github.com/kietle/zenreply/pkg/response"
)

// GetMe godoc
//
//	@Summary		Get current user profile
//	@Description	Returns the authenticated user's profile information
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=model.User}
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/users/me [get]
func (h *Handler) GetMe(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	user, err := h.authService.GetUserByID(ctx, userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.OK(c, "user profile retrieved", user)
}

// DeleteMe godoc
//
//	@Summary		Deactivate current user account
//	@Description	Deactivates the authenticated user's account and revokes access
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/users/me [delete]
func (h *Handler) DeleteMe(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	if err := h.authService.DeactivateUser(ctx, userID); err != nil {
		response.InternalServerError(c, "failed to deactivate account")
		return
	}

	response.OK(c, "account deactivated successfully", nil)
}
