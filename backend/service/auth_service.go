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

const oauthStatePrefix = "oauth_state:"
const oauthStateTTL = 10 * time.Minute

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

// GenerateOAuthState creates a cryptographically random state token and stores it in Redis.
func (s *AuthService) GenerateOAuthState(ctx context.Context) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("authService.GenerateOAuthState: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)

	key := oauthStatePrefix + state
	if err := s.rdb.Set(ctx, key, "1", oauthStateTTL).Err(); err != nil {
		return "", fmt.Errorf("authService.GenerateOAuthState: store state: %w", err)
	}

	return state, nil
}

// ValidateOAuthState verifies the state token and removes it from Redis (one-time use).
func (s *AuthService) ValidateOAuthState(ctx context.Context, state string) error {
	key := oauthStatePrefix + state
	result, err := s.rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) || result == "" {
		return errors.New("invalid or expired oauth state")
	}
	if err != nil {
		return fmt.Errorf("authService.ValidateOAuthState: %w", err)
	}
	return nil
}

// BuildAuthURL returns the Slack OAuth URL with a fresh state token.
func (s *AuthService) BuildAuthURL(ctx context.Context) (string, string, error) {
	state, err := s.GenerateOAuthState(ctx)
	if err != nil {
		return "", "", err
	}
	url := s.oauthSvc.BuildAuthURL(state)
	return url, state, nil
}

// HandleCallback processes the Slack OAuth callback, upserts the user, and returns a JWT.
func (s *AuthService) HandleCallback(ctx context.Context, code, state string) (*model.User, string, error) {
	if err := s.ValidateOAuthState(ctx, state); err != nil {
		return nil, "", err
	}

	oauthResult, err := s.oauthSvc.ExchangeCode(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("authService.HandleCallback: exchange code: %w", err)
	}

	user := &model.User{
		SlackUserID: oauthResult.SlackUserID,
		SlackTeamID: oauthResult.SlackTeamID,
		SlackName:   oauthResult.SlackName,
		Email:       oauthResult.Email,
		AvatarURL:   oauthResult.AvatarURL,
		AccessToken: oauthResult.AccessToken,
		BotToken:    oauthResult.BotToken,
		TokenScope:  oauthResult.TokenScope,
	}

	savedUser, err := s.userRepo.Upsert(ctx, user)
	if err != nil {
		return nil, "", fmt.Errorf("authService.HandleCallback: upsert user: %w", err)
	}

	// Ensure default settings exist for new users.
	_, err = s.settingsRepo.FindByUserID(ctx, savedUser.ID)
	if errors.Is(err, repository.ErrNotFound) {
		defaultSettings := &model.UserSettings{
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
		if _, err := s.settingsRepo.Upsert(ctx, defaultSettings); err != nil {
			return nil, "", fmt.Errorf("authService.HandleCallback: create default settings: %w", err)
		}
	}

	token, err := s.GenerateJWT(savedUser)
	if err != nil {
		return nil, "", fmt.Errorf("authService.HandleCallback: generate jwt: %w", err)
	}

	return savedUser, token, nil
}

// GenerateJWT creates a signed JWT for the given user.
func (s *AuthService) GenerateJWT(user *model.User) (string, error) {
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
		return "", fmt.Errorf("authService.GenerateJWT: %w", err)
	}
	return signed, nil
}

// GetUserByID retrieves a user by their internal UUID.
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// DeactivateUser marks a user as inactive.
func (s *AuthService) DeactivateUser(ctx context.Context, id string) error {
	return s.userRepo.Deactivate(ctx, id)
}
