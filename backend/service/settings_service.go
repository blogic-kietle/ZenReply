// Package service contains the business logic layer for ZenReply.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kietle/zenreply/model"
	"github.com/kietle/zenreply/repository"
)

// SettingsService manages user settings business logic.
type SettingsService struct {
	settingsRepo repository.SettingsRepository
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(settingsRepo repository.SettingsRepository) *SettingsService {
	return &SettingsService{settingsRepo: settingsRepo}
}

// GetSettings retrieves settings for a user, creating defaults if none exist.
func (s *SettingsService) GetSettings(ctx context.Context, userID string) (*model.UserSettings, error) {
	settings, err := s.settingsRepo.FindByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
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
		return s.settingsRepo.Upsert(ctx, defaults)
	}
	if err != nil {
		return nil, fmt.Errorf("settingsService.GetSettings: %w", err)
	}
	return settings, nil
}

// UpdateSettings saves updated settings for a user.
func (s *SettingsService) UpdateSettings(ctx context.Context, settings *model.UserSettings) (*model.UserSettings, error) {
	if settings.CooldownMinutes < 1 {
		settings.CooldownMinutes = 1
	}
	if settings.CooldownMinutes > 60 {
		settings.CooldownMinutes = 60
	}
	if settings.DefaultMessage == "" {
		settings.DefaultMessage = "I am currently in a deep work session and will reply as soon as I am available."
	}

	result, err := s.settingsRepo.Upsert(ctx, settings)
	if err != nil {
		return nil, fmt.Errorf("settingsService.UpdateSettings: %w", err)
	}
	return result, nil
}

// AddToWhitelist adds a Slack user ID to the whitelist.
func (s *SettingsService) AddToWhitelist(ctx context.Context, userID, slackID string) error {
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}
	if contains(settings.Whitelist, slackID) {
		return nil
	}
	// Remove from blacklist if present.
	settings.Blacklist = removeFrom(settings.Blacklist, slackID)
	settings.Whitelist = append(settings.Whitelist, slackID)
	_, err = s.settingsRepo.Upsert(ctx, settings)
	return err
}

// RemoveFromWhitelist removes a Slack user ID from the whitelist.
func (s *SettingsService) RemoveFromWhitelist(ctx context.Context, userID, slackID string) error {
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}
	settings.Whitelist = removeFrom(settings.Whitelist, slackID)
	_, err = s.settingsRepo.Upsert(ctx, settings)
	return err
}

// AddToBlacklist adds a Slack user ID to the blacklist.
func (s *SettingsService) AddToBlacklist(ctx context.Context, userID, slackID string) error {
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}
	if contains(settings.Blacklist, slackID) {
		return nil
	}
	// Remove from whitelist if present.
	settings.Whitelist = removeFrom(settings.Whitelist, slackID)
	settings.Blacklist = append(settings.Blacklist, slackID)
	_, err = s.settingsRepo.Upsert(ctx, settings)
	return err
}

// RemoveFromBlacklist removes a Slack user ID from the blacklist.
func (s *SettingsService) RemoveFromBlacklist(ctx context.Context, userID, slackID string) error {
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}
	settings.Blacklist = removeFrom(settings.Blacklist, slackID)
	_, err = s.settingsRepo.Upsert(ctx, settings)
	return err
}

func removeFrom(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
