// Package handler provides HTTP request handlers for the ZenReply API.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kietle/zenreply/pkg/response"
)

// SlackAuthURLResponse contains the Slack OAuth authorization URL.
type SlackAuthURLResponse struct {
	URL   string `json:"url" example:"https://slack.com/oauth/v2/authorize?..."`
	State string `json:"state" example:"abc123"`
}

// SlackAuthCallbackResponse contains the result of a successful OAuth flow.
type SlackAuthCallbackResponse struct {
	Token    string      `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	UserID   string      `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SlackID  string      `json:"slack_id" example:"U01234567"`
	Name     string      `json:"name" example:"John Doe"`
	Email    string      `json:"email" example:"john@example.com"`
	Avatar   string      `json:"avatar" example:"https://avatars.slack-edge.com/..."`
}

// SlackAuthURL godoc
//
//	@Summary		Get Slack OAuth URL
//	@Description	Returns the Slack OAuth authorization URL for user login
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.Response{data=SlackAuthURLResponse}
//	@Failure		500	{object}	response.Response
//	@Router			/slack/auth [get]
func (h *Handler) SlackAuthURL(c *gin.Context) {
	ctx := c.Request.Context()

	url, state, err := h.authService.BuildAuthURL(ctx)
	if err != nil {
		response.InternalServerError(c, "failed to generate oauth url")
		return
	}

	response.OK(c, "oauth url generated", SlackAuthURLResponse{
		URL:   url,
		State: state,
	})
}

// SlackCallback godoc
//
//	@Summary		Handle Slack OAuth callback
//	@Description	Exchanges the OAuth code for tokens, upserts the user, and returns a JWT
//	@Tags			auth
//	@Produce		json
//	@Param			code	query	string	true	"OAuth authorization code from Slack"
//	@Param			state	query	string	true	"State token for CSRF protection"
//	@Success		200		{object}	response.Response{data=SlackAuthCallbackResponse}
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		500		{object}	response.Response
//	@Router			/slack/callback [get]
func (h *Handler) SlackCallback(c *gin.Context) {
	ctx := c.Request.Context()

	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		response.BadRequest(c, "MISSING_PARAMS", "code and state query parameters are required", "")
		return
	}

	user, token, err := h.authService.HandleCallback(ctx, code, state)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.OK(c, "authentication successful", SlackAuthCallbackResponse{
		Token:   token,
		UserID:  user.ID,
		SlackID: user.SlackUserID,
		Name:    user.SlackName,
		Email:   user.Email,
		Avatar:  user.AvatarURL,
	})
}

// SlackCallbackRedirect godoc
//
//	@Summary		Handle Slack OAuth callback with redirect
//	@Description	Same as /slack/callback but redirects to the frontend with the token as a query param
//	@Tags			auth
//	@Param			code	query	string	true	"OAuth authorization code from Slack"
//	@Param			state	query	string	true	"State token for CSRF protection"
//	@Success		302
//	@Failure		400	{object}	response.Response
//	@Router			/slack/callback/redirect [get]
func (h *Handler) SlackCallbackRedirect(c *gin.Context) {
	ctx := c.Request.Context()

	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		response.BadRequest(c, "MISSING_PARAMS", "code and state are required", "")
		return
	}

	_, token, err := h.authService.HandleCallback(ctx, code, state)
	if err != nil {
		c.Redirect(http.StatusFound, h.cfg.App.FrontendURL+"/auth/error?message="+err.Error())
		return
	}

	c.Redirect(http.StatusFound, h.cfg.App.FrontendURL+"/auth/callback?token="+token)
}
