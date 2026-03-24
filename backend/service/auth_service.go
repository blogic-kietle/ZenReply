// Package service contains the business logic layer for ZenReply.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kietle/zenreply/config"
	"github.com/kietle/zenreply/model"
	"github.com/kietle/zenreply/pkg/middleware"
	slackpkg "github.com/kietle/zenreply/pkg/slack"
	"github.com/kietle/zenreply/repository"
	"github.com/redis/go-redis/v9"
)

const (
	oauthStatePrefix = "oauth_state:"
	oauthStateTTL    = 10 * time.Minute
)

// AuthService handles authentication and Slack OAuth flows.
type AuthService struct {
	cfg          *config.Config
	userRepo     repository.UserRepository
	settingsRepo repository.SettingsRepository
	oauthSvc     *slackpkg.OAuthService
	rdb          *redis.Client
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	cfg *config.Config,
	userRepo repository.UserRepository,
	settingsRepo repository.SettingsRepository,
	oauthSvc *slackpkg.OAuthService,
	rdb *redis.Client,
) *AuthService {
	return &AuthService{
		cfg:          cfg,
		userRepo:     userRepo,
		settingsRepo: settingsRepo,
		oauthSvc:     oauthSvc,
		rdb:          rdb,
	}
}

// BuildAuthURL returns the Slack OAuth URL with a fresh CSRF state token.
func (s *AuthService) BuildAuthURL(ctx context.Context) (url, state string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("authService.BuildAuthURL: generate state: %w", err)
	}
	state = base64.URLEncoding.EncodeToString(b)

	if err = s.rdb.Set(ctx, oauthStatePrefix+state, "1", oauthStateTTL).Err(); err != nil {
		return "", "", fmt.Errorf("authService.BuildAuthURL: store state: %w", err)
	}

	url = s.oauthSvc.BuildAuthURL(state)
	return url, state, nil
}

// HandleCallback validates the state, exchanges the code for a User Token,
// upserts the user, creates default settings if new, and returns a JWT.
func (s *AuthService) HandleCallback(ctx context.Context, code, state string) (*model.User, string, error) {
	// Validate CSRF state (one-time use).
	result, err := s.rdb.GetDel(ctx, oauthStatePrefix+state).Result()
	if errors.Is(err, redis.Nil) || result == "" {
		return nil, "", errors.New("invalid or expired oauth state")
	}
	if err != nil {
		return nil, "", fmt.Errorf("authService.HandleCallback: validate state: %w", err)
	}

	// Exchange code for User Token (xoxp-...).
	oauthResult, err := s.oauthSvc.ExchangeCode(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("authService.HandleCallback: %w", err)
	}

	// Upsert user — access_token is the User Token.
	user := &model.User{
		SlackUserID: oauthResult.SlackUserID,
		SlackTeamID: oauthResult.SlackTeamID,
		SlackName:   oauthResult.SlackName,
		Email:       oauthResult.Email,
		AvatarURL:   oauthResult.AvatarURL,
		AccessToken: oauthResult.AccessToken,
		TokenScope:  oauthResult.TokenScope,
	}
	savedUser, err := s.userRepo.Upsert(ctx, user)
	if err != nil {
		return nil, "", fmt.Errorf("authService.HandleCallback: upsert user: %w", err)
	}

	// Create default settings for first-time users.
	_, err = s.settingsRepo.FindByUserID(ctx, savedUser.ID)
	if errors.Is(err, repository.ErrNotFound) {
		defaults := &model.UserSettings{
			UserID:           savedUser.ID,
			DefaultMessage:   "I am currently in a deep work session and will reply as soon as I am available. Thank you for your patience.",
			DefaultReason:    "Deep Work",
			CooldownMinutes:  3,
			Whitelist:        []string{},
			Blacklist:        []string{},
			ReplyInThread:    true,
			NotifyOnResume:   false,
			AutoReplyEnabled: true,
		}
		if _, err := s.settingsRepo.Upsert(ctx, defaults); err != nil {
			return nil, "", fmt.Errorf("authService.HandleCallback: create default settings: %w", err)
		}
	}

	token, err := s.generateJWT(savedUser)
	if err != nil {
		return nil, "", fmt.Errorf("authService.HandleCallback: %w", err)
	}

	return savedUser, token, nil
}

// GetUserByID retrieves a user by their internal UUID.
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// DeactivateUser marks a user as inactive.
func (s *AuthService) DeactivateUser(ctx context.Context, id string) error {
	return s.userRepo.Deactivate(ctx, id)
}

func (s *AuthService) generateJWT(user *model.User) (string, error) {
	now := time.Now()
	claims := &middleware.Claims{
		UserID:      user.ID,
		SlackUserID: user.SlackUserID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWT.Expiration)),
			Issuer:    s.cfg.App.Name,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("authService.generateJWT: %w", err)
	}
	return signed, nil
}
