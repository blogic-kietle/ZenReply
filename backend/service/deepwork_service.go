// Package service contains the business logic layer for ZenReply.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kietle/zenreply/model"
	"github.com/kietle/zenreply/repository"
	slackpkg "github.com/kietle/zenreply/pkg/slack"
	"github.com/redis/go-redis/v9"
)

const (
	deepWorkKeyPrefix = "deepwork:active:"
	cooldownKeyPrefix = "deepwork:cooldown:"
)

// DeepWorkService manages deep work sessions and the auto-reply engine.
type DeepWorkService struct {
	sessionRepo    repository.SessionRepository
	settingsRepo   repository.SettingsRepository
	userRepo       repository.UserRepository
	messageLogRepo repository.MessageLogRepository
	rdb            *redis.Client
	messenger      *slackpkg.Messenger
	logger         *slog.Logger
}

// NewDeepWorkService creates a new DeepWorkService.
func NewDeepWorkService(
	sessionRepo repository.SessionRepository,
	settingsRepo repository.SettingsRepository,
	userRepo repository.UserRepository,
	messageLogRepo repository.MessageLogRepository,
	rdb *redis.Client,
	messenger *slackpkg.Messenger,
	logger *slog.Logger,
) *DeepWorkService {
	return &DeepWorkService{
		sessionRepo:    sessionRepo,
		settingsRepo:   settingsRepo,
		userRepo:       userRepo,
		messageLogRepo: messageLogRepo,
		rdb:            rdb,
		messenger:      messenger,
		logger:         logger,
	}
}

// StartSession begins a new deep work session and caches the status in Redis.
func (s *DeepWorkService) StartSession(ctx context.Context, userID, reason string) (*model.DeepWorkSession, error) {
	session, err := s.sessionRepo.Create(ctx, userID, reason)
	if err != nil {
		return nil, fmt.Errorf("deepWorkService.StartSession: %w", err)
	}

	statusData, _ := json.Marshal(map[string]any{
		"session_id": session.ID,
		"reason":     session.Reason,
		"started_at": session.StartTime.Format(time.RFC3339Nano),
	})
	if err := s.rdb.Set(ctx, deepWorkKeyPrefix+userID, statusData, 0).Err(); err != nil {
		s.logger.Warn("failed to cache deep work status", slog.String("error", err.Error()))
	}

	s.logger.Info("deep work session started",
		slog.String("user_id", userID),
		slog.String("session_id", session.ID),
	)
	return session, nil
}

// EndSession terminates the active session and clears the Redis cache.
func (s *DeepWorkService) EndSession(ctx context.Context, userID string) (*model.DeepWorkSession, error) {
	session, err := s.sessionRepo.FindActiveByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errors.New("no active deep work session found")
	}
	if err != nil {
		return nil, fmt.Errorf("deepWorkService.EndSession: %w", err)
	}

	if err := s.sessionRepo.End(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("deepWorkService.EndSession: end: %w", err)
	}

	s.rdb.Del(ctx, deepWorkKeyPrefix+userID)

	s.logger.Info("deep work session ended",
		slog.String("user_id", userID),
		slog.String("session_id", session.ID),
	)
	return session, nil
}

// GetStatus returns the current deep work status, preferring Redis cache.
func (s *DeepWorkService) GetStatus(ctx context.Context, userID string) (*model.DeepWorkStatus, error) {
	data, err := s.rdb.Get(ctx, deepWorkKeyPrefix+userID).Bytes()
	if errors.Is(err, redis.Nil) {
		return &model.DeepWorkStatus{IsActive: false}, nil
	}
	if err != nil {
		// Fallback to DB on Redis error.
		session, dbErr := s.sessionRepo.FindActiveByUserID(ctx, userID)
		if errors.Is(dbErr, repository.ErrNotFound) {
			return &model.DeepWorkStatus{IsActive: false}, nil
		}
		if dbErr != nil {
			return nil, fmt.Errorf("deepWorkService.GetStatus: %w", dbErr)
		}
		return &model.DeepWorkStatus{
			IsActive:  true,
			SessionID: session.ID,
			Reason:    session.Reason,
			StartedAt: &session.StartTime,
		}, nil
	}

	var cached map[string]any
	if err := json.Unmarshal(data, &cached); err != nil {
		return &model.DeepWorkStatus{IsActive: false}, nil
	}

	status := &model.DeepWorkStatus{IsActive: true}
	if v, ok := cached["session_id"].(string); ok {
		status.SessionID = v
	}
	if v, ok := cached["reason"].(string); ok {
		status.Reason = v
	}
	if v, ok := cached["started_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			status.StartedAt = &t
		}
	}
	return status, nil
}

// ListSessions returns a paginated list of sessions for a user.
func (s *DeepWorkService) ListSessions(ctx context.Context, userID string, limit, offset int) ([]*model.DeepWorkSession, int64, error) {
	return s.sessionRepo.ListByUserID(ctx, userID, limit, offset)
}

// GetSessionByID retrieves a single session by its ID.
func (s *DeepWorkService) GetSessionByID(ctx context.Context, id string) (*model.DeepWorkSession, error) {
	return s.sessionRepo.FindByID(ctx, id)
}

// ListMessageLogs returns paginated message logs for a user.
func (s *DeepWorkService) ListMessageLogs(ctx context.Context, userID string, limit, offset int) ([]*model.MessageLog, int64, error) {
	return s.messageLogRepo.ListByUserID(ctx, userID, limit, offset)
}

// ListSessionMessageLogs returns all message logs for a specific session.
func (s *DeepWorkService) ListSessionMessageLogs(ctx context.Context, sessionID string) ([]*model.MessageLog, error) {
	return s.messageLogRepo.ListBySessionID(ctx, sessionID)
}

// HandleIncomingMessage is the core auto-reply engine.
// It is called when the Slack Events API reports a DM sent to one of our users.
//
// ownerSlackID is the Slack user ID of the ZenReply user who received the message.
// senderSlackID is the Slack user ID of the person who sent the message.
//
// The function uses the owner's User Token (xoxp-) to reply — so the message
// appears to come from the owner themselves, not from a bot.
func (s *DeepWorkService) HandleIncomingMessage(
	ctx context.Context,
	ownerSlackID, senderSlackID, channelID, messageText, originalTS, threadTS string,
) error {
	// Skip self-messages.
	if ownerSlackID == senderSlackID {
		return nil
	}

	// Look up the ZenReply user who owns this Slack account.
	owner, err := s.userRepo.FindBySlackUserID(ctx, ownerSlackID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil // Not a ZenReply user.
	}
	if err != nil {
		return fmt.Errorf("deepWorkService.HandleIncomingMessage: find owner: %w", err)
	}

	// Check if the owner is in an active deep work session.
	status, err := s.GetStatus(ctx, owner.ID)
	if err != nil || !status.IsActive {
		return nil
	}

	// Load settings.
	settings, err := s.settingsRepo.FindByUserID(ctx, owner.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deepWorkService.HandleIncomingMessage: find settings: %w", err)
	}
	if !settings.AutoReplyEnabled {
		return nil
	}

	// Blacklist check — never auto-reply to these senders.
	if contains(settings.Blacklist, senderSlackID) {
		s.logger.Info("sender blacklisted, skipping auto-reply",
			slog.String("sender", senderSlackID),
			slog.String("owner", ownerSlackID),
		)
		return nil
	}

	// Cooldown check — avoid spamming the same sender.
	cooldownKey := cooldownKeyPrefix + owner.ID + ":" + senderSlackID
	if n, _ := s.rdb.Exists(ctx, cooldownKey).Result(); n > 0 {
		s.logger.Info("sender in cooldown, skipping auto-reply",
			slog.String("sender", senderSlackID),
			slog.String("owner", ownerSlackID),
		)
		return nil
	}

	// Determine reply thread.
	replyThreadTS := ""
	if settings.ReplyInThread {
		if threadTS != "" {
			replyThreadTS = threadTS
		} else {
			replyThreadTS = originalTS
		}
	}

	// Send auto-reply using the owner's own User Token (xoxp-).
	// The recipient sees the message as coming from the owner themselves.
	autoReplyText := settings.DefaultMessage
	sendErr := s.messenger.SendAutoReply(ctx, owner.AccessToken, channelID, autoReplyText, replyThreadTS)

	// Log the attempt regardless of outcome.
	logEntry := &model.MessageLog{
		UserID:        owner.ID,
		SessionID:     status.SessionID,
		SenderSlackID: senderSlackID,
		ChannelID:     channelID,
		OriginalTS:    originalTS,
		ThreadTS:      threadTS,
		MessageText:   messageText,
		AutoReplyText: autoReplyText,
		WasSent:       sendErr == nil,
	}
	if sendErr != nil {
		logEntry.ErrorMessage = sendErr.Error()
	}
	if _, logErr := s.messageLogRepo.Create(ctx, logEntry); logErr != nil {
		s.logger.Warn("failed to write message log", slog.String("error", logErr.Error()))
	}

	if sendErr != nil {
		return fmt.Errorf("deepWorkService.HandleIncomingMessage: send reply: %w", sendErr)
	}

	// Set cooldown to prevent reply spam.
	cooldownDur := time.Duration(settings.CooldownMinutes) * time.Minute
	if err := s.rdb.Set(ctx, cooldownKey, "1", cooldownDur).Err(); err != nil {
		s.logger.Warn("failed to set cooldown", slog.String("error", err.Error()))
	}

	s.logger.Info("auto-reply sent",
		slog.String("owner", ownerSlackID),
		slog.String("sender", senderSlackID),
		slog.String("channel", channelID),
	)
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
