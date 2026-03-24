// Package slack provides Slack API integration utilities for ZenReply.
package slack

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kietle/zenreply/config"
	slacklib "github.com/slack-go/slack"
)

// userScopes are the Slack User Token scopes ZenReply requires.
// These are requested as user_scope (not scope) so the resulting token is
// a User Token (xoxp-...) that acts as the user themselves.
const userScopes = "chat:write,im:history,im:read,mpim:history,mpim:read,channels:history,groups:history,users:read,users:read.email"

// OAuthService handles the Slack OAuth 2.0 "Sign in with Slack" flow.
type OAuthService struct {
	cfg *config.SlackConfig
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(cfg *config.SlackConfig) *OAuthService {
	return &OAuthService{cfg: cfg}
}

// OAuthResult holds the result of a successful OAuth token exchange.
type OAuthResult struct {
	// AccessToken is the Slack User Token (xoxp-...) for the authorizing user.
	AccessToken string
	TokenScope  string
	SlackUserID string
	SlackTeamID string
	SlackName   string
	Email       string
	AvatarURL   string
}

// BuildAuthURL returns the Slack OAuth authorization URL requesting user_scope only.
// No bot scope is requested — ZenReply acts as the user, not a bot.
func (s *OAuthService) BuildAuthURL(state string) string {
	return fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&user_scope=%s&redirect_uri=%s&state=%s",
		s.cfg.ClientID,
		userScopes,
		s.cfg.RedirectURL,
		state,
	)
}

// ExchangeCode exchanges an OAuth code for a User Token and fetches the user's profile.
func (s *OAuthService) ExchangeCode(ctx context.Context, code string) (*OAuthResult, error) {
	resp, err := slacklib.GetOAuthV2ResponseContext(
		ctx,
		http.DefaultClient,
		s.cfg.ClientID,
		s.cfg.ClientSecret,
		code,
		s.cfg.RedirectURL,
	)
	if err != nil {
		return nil, fmt.Errorf("slack oauth exchange: %w", err)
	}
	if !resp.Ok {
		return nil, fmt.Errorf("slack oauth exchange error: %s", resp.Error)
	}

	// When only user_scope is requested, the user token lives in AuthedUser.
	userToken := resp.AuthedUser.AccessToken
	if userToken == "" {
		return nil, fmt.Errorf("slack oauth: no user access token in response (check user_scope configuration)")
	}

	result := &OAuthResult{
		AccessToken: userToken,
		TokenScope:  resp.AuthedUser.Scope,
		SlackUserID: resp.AuthedUser.ID,
		SlackTeamID: resp.Team.ID,
	}

	// Fetch the user's profile using their own token.
	userClient := slacklib.New(userToken)
	userInfo, err := userClient.GetUserInfoContext(ctx, resp.AuthedUser.ID)
	if err == nil && userInfo != nil {
		result.SlackName = userInfo.RealName
		result.Email = userInfo.Profile.Email
		result.AvatarURL = userInfo.Profile.Image192
	}

	return result, nil
}
