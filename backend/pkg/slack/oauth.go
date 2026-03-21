// Package slack provides Slack API integration utilities for ZenReply.
package slack

import (
	"context"
	"fmt"
	"net/http"

	slacklib "github.com/slack-go/slack"
	"github.com/kietle/zenreply/config"
)

// OAuthService handles the Slack OAuth 2.0 flow.
type OAuthService struct {
	cfg *config.SlackConfig
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(cfg *config.SlackConfig) *OAuthService {
	return &OAuthService{cfg: cfg}
}

// OAuthResult holds the result of a successful OAuth token exchange.
type OAuthResult struct {
	AccessToken string
	BotToken    string
	TokenScope  string
	SlackUserID string
	SlackTeamID string
	SlackName   string
	Email       string
	AvatarURL   string
}

// BuildAuthURL returns the Slack OAuth authorization URL.
func (s *OAuthService) BuildAuthURL(state string) string {
	return fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&scope=%s&user_scope=%s&redirect_uri=%s&state=%s",
		s.cfg.ClientID,
		"chat:write,im:history,im:read,channels:history,groups:history",
		"chat:write,im:history,im:read,channels:history,groups:history,users:read,users:read.email",
		s.cfg.RedirectURL,
		state,
	)
}

// ExchangeCode exchanges an OAuth code for access tokens and user info.
func (s *OAuthService) ExchangeCode(ctx context.Context, code string) (*OAuthResult, error) {
	resp, err := slacklib.GetOAuthV2ResponseContext(ctx, http.DefaultClient, s.cfg.ClientID, s.cfg.ClientSecret, code, s.cfg.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("slack oauth exchange: %w", err)
	}

	if !resp.Ok {
		return nil, fmt.Errorf("slack oauth exchange error: %s", resp.Error)
	}

	result := &OAuthResult{
		AccessToken: resp.AuthedUser.AccessToken,
		BotToken:    resp.AccessToken,
		TokenScope:  resp.Scope,
		SlackUserID: resp.AuthedUser.ID,
		SlackTeamID: resp.Team.ID,
	}

	// Fetch user profile using the user access token.
	if resp.AuthedUser.AccessToken != "" {
		userClient := slacklib.New(resp.AuthedUser.AccessToken)
		userInfo, err := userClient.GetUserInfoContext(ctx, resp.AuthedUser.ID)
		if err == nil && userInfo != nil {
			result.SlackName = userInfo.RealName
			result.Email = userInfo.Profile.Email
			result.AvatarURL = userInfo.Profile.Image192
		}
	}

	return result, nil
}
