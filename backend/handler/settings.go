// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/model"
	"github.com/kietle/zenreply/pkg/middleware"
	"github.com/kietle/zenreply/pkg/response"
)

// UpdateSettingsRequest is the request body for updating user settings.
type UpdateSettingsRequest struct {
	DefaultMessage   string `json:"default_message" example:"I am in a deep work session, will reply soon."`
	DefaultReason    string `json:"default_reason" example:"Deep Work"`
	CooldownMinutes  int    `json:"cooldown_minutes" example:"3"`
	ReplyInThread    bool   `json:"reply_in_thread" example:"true"`
	NotifyOnResume   bool   `json:"notify_on_resume" example:"false"`
	AutoReplyEnabled bool   `json:"auto_reply_enabled" example:"true"`
}

// ListEntryRequest is the request body for adding an entry to whitelist/blacklist.
type ListEntryRequest struct {
	SlackUserID string `json:"slack_user_id" binding:"required" example:"U01234567"`
}

// GetSettings godoc
//
//	@Summary		Get user settings
//	@Description	Returns the authenticated user's auto-reply configuration
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=model.UserSettings}
//	@Failure		401	{object}	response.Response
//	@Router			/settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	settings, err := h.settingsService.GetSettings(ctx, userID)
	if err != nil {
		response.InternalServerError(c, "failed to retrieve settings")
		return
	}

	response.OK(c, "settings retrieved", settings)
}

// UpdateSettings godoc
//
//	@Summary		Update user settings
//	@Description	Updates the authenticated user's auto-reply configuration
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateSettingsRequest	true	"Settings to update"
//	@Success		200		{object}	response.Response{data=model.UserSettings}
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		500		{object}	response.Response
//	@Router			/settings [put]
func (h *Handler) UpdateSettings(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "invalid request body", err.Error())
		return
	}

	existing, err := h.settingsService.GetSettings(ctx, userID)
	if err != nil {
		response.InternalServerError(c, "failed to retrieve existing settings")
		return
	}

	// Merge request fields into existing settings.
	if req.DefaultMessage != "" {
		existing.DefaultMessage = req.DefaultMessage
	}
	if req.DefaultReason != "" {
		existing.DefaultReason = req.DefaultReason
	}
	if req.CooldownMinutes > 0 {
		existing.CooldownMinutes = req.CooldownMinutes
	}
	existing.ReplyInThread = req.ReplyInThread
	existing.NotifyOnResume = req.NotifyOnResume
	existing.AutoReplyEnabled = req.AutoReplyEnabled

	updated, err := h.settingsService.UpdateSettings(ctx, existing)
	if err != nil {
		response.InternalServerError(c, "failed to update settings")
		return
	}

	response.OK(c, "settings updated", updated)
}

// GetWhitelist godoc
//
//	@Summary		Get whitelist
//	@Description	Returns the list of Slack user IDs that always receive auto-replies
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=[]string}
//	@Failure		401	{object}	response.Response
//	@Router			/settings/whitelist [get]
func (h *Handler) GetWhitelist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	settings, err := h.settingsService.GetSettings(ctx, userID)
	if err != nil {
		response.InternalServerError(c, "failed to retrieve whitelist")
		return
	}

	response.OK(c, "whitelist retrieved", settings.Whitelist)
}

// AddToWhitelist godoc
//
//	@Summary		Add to whitelist
//	@Description	Adds a Slack user ID to the whitelist. The user is also removed from the blacklist if present.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		ListEntryRequest	true	"Slack user ID to add"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/settings/whitelist [post]
func (h *Handler) AddToWhitelist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	var req ListEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "slack_user_id is required", err.Error())
		return
	}

	if err := h.settingsService.AddToWhitelist(ctx, userID, req.SlackUserID); err != nil {
		response.InternalServerError(c, "failed to add to whitelist")
		return
	}

	response.OK(c, "user added to whitelist", nil)
}

// RemoveFromWhitelist godoc
//
//	@Summary		Remove from whitelist
//	@Description	Removes a Slack user ID from the whitelist
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slack_user_id	path	string	true	"Slack User ID to remove"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/settings/whitelist/{slack_user_id} [delete]
func (h *Handler) RemoveFromWhitelist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)
	slackUserID := c.Param("slack_user_id")

	if err := h.settingsService.RemoveFromWhitelist(ctx, userID, slackUserID); err != nil {
		response.InternalServerError(c, "failed to remove from whitelist")
		return
	}

	response.OK(c, "user removed from whitelist", nil)
}

// GetBlacklist godoc
//
//	@Summary		Get blacklist
//	@Description	Returns the list of Slack user IDs that never receive auto-replies
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=[]string}
//	@Failure		401	{object}	response.Response
//	@Router			/settings/blacklist [get]
func (h *Handler) GetBlacklist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	settings, err := h.settingsService.GetSettings(ctx, userID)
	if err != nil {
		response.InternalServerError(c, "failed to retrieve blacklist")
		return
	}

	response.OK(c, "blacklist retrieved", settings.Blacklist)
}

// AddToBlacklist godoc
//
//	@Summary		Add to blacklist
//	@Description	Adds a Slack user ID to the blacklist. The user is also removed from the whitelist if present.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		ListEntryRequest	true	"Slack user ID to add"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/settings/blacklist [post]
func (h *Handler) AddToBlacklist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	var req ListEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "slack_user_id is required", err.Error())
		return
	}

	if err := h.settingsService.AddToBlacklist(ctx, userID, req.SlackUserID); err != nil {
		response.InternalServerError(c, "failed to add to blacklist")
		return
	}

	response.OK(c, "user added to blacklist", nil)
}

// RemoveFromBlacklist godoc
//
//	@Summary		Remove from blacklist
//	@Description	Removes a Slack user ID from the blacklist
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slack_user_id	path	string	true	"Slack User ID to remove"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/settings/blacklist/{slack_user_id} [delete]
func (h *Handler) RemoveFromBlacklist(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)
	slackUserID := c.Param("slack_user_id")

	if err := h.settingsService.RemoveFromBlacklist(ctx, userID, slackUserID); err != nil {
		response.InternalServerError(c, "failed to remove from blacklist")
		return
	}

	response.OK(c, "user removed from blacklist", nil)
}

// ResetSettings godoc
//
//	@Summary		Reset settings to defaults
//	@Description	Resets all user settings to their default values
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=model.UserSettings}
//	@Failure		401	{object}	response.Response
//	@Failure		500	{object}	response.Response
//	@Router			/settings/reset [post]
func (h *Handler) ResetSettings(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(middleware.ContextKeyUserID)

	defaults := &model.UserSettings{
		UserID:           userID,
		DefaultMessage:   "I am currently in a deep work session and will reply as soon as I am available.",
		DefaultReason:    "Deep Work",
		CooldownMinutes:  3,
		Whitelist:        []string{},
		Blacklist:        []string{},
		ReplyInThread:    true,
		NotifyOnResume:   false,
		AutoReplyEnabled: true,
	}

	updated, err := h.settingsService.UpdateSettings(ctx, defaults)
	if err != nil {
		response.InternalServerError(c, "failed to reset settings")
		return
	}

	response.OK(c, "settings reset to defaults", updated)
}
