// Package service contains the business logic layer for ZenReply.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	slackpkg "github.com/kietle/zenreply/pkg/slack"
	"github.com/kietle/zenreply/model"
	"github.com/kietle/zenreply/repository"
	"github.com/redis/go-redis/v9"
)

const (
	deepWorkKeyPrefix     = "deepwork:active:"
	cooldownKeyPrefix     = "deepwork:cooldown:"
)

// DeepWorkService manages deep work sessions, timers, and auto-reply logic.
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

// StartSession begins a new deep work session for the user.
func (s *DeepWorkService) StartSession(ctx context.Context, userID, reason string) (*model.DeepWorkSession, error) {
	session, err := s.sessionRepo.Create(ctx, userID, reason)
	if err != nil {
		return nil, fmt.Errorf("deepWorkService.StartSession: %w", err)
	}

	statusData, _ := json.Marshal(map[string]interface{}{
		"session_id": session.ID,
		"reason":     session.Reason,
		"started_at": session.StartTime.Format(time.RFC3339Nano),
	})
	key := deepWorkKeyPrefix + userID
	if err := s.rdb.Set(ctx, key, statusData, 0).Err(); err != nil {
		s.logger.Warn("failed to cache deep work status in redis",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.Info("deep work session started",
		slog.String("user_id", userID),
		slog.String("session_id", session.ID),
		slog.String("reason", reason),
	)
	return session, nil
}

// EndSession terminates the active deep work session for the user.
func (s *DeepWorkService) EndSession(ctx context.Context, userID string) (*model.DeepWorkSession, error) {
	session, err := s.sessionRepo.FindActiveByUserID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errors.New("no active deep work session found")
	}
	if err != nil {
		return nil, fmt.Errorf("deepWorkService.EndSession: find session: %w", err)
	}

	if err := s.sessionRepo.End(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("deepWorkService.EndSession: end session: %w", err)
	}

	s.rdb.Del(ctx, deepWorkKeyPrefix+userID)

	s.logger.Info("deep work session ended",
		slog.String("user_id", userID),
		slog.String("session_id", session.ID),
	)
	return session, nil
}

// GetStatus returns the current deep work status for a user.
func (s *DeepWorkService) GetStatus(ctx context.Context, userID string) (*model.DeepWorkStatus, error) {
	key := deepWorkKeyPrefix + userID
	data, err := s.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return &model.DeepWorkStatus{IsActive: false}, nil
	}
	if err != nil {
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

	var cached map[string]interface{}
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

// HandleIncomingMessage processes an incoming Slack message and sends an auto-reply if needed.
func (s *DeepWorkService) HandleIncomingMessage(
	ctx context.Context,
	ownerSlackID, senderSlackID, channelID, messageText, originalTS, threadTS string,
) error {
	owner, err := s.userRepo.FindBySlackUserID(ctx, ownerSlackID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deepWorkService.HandleIncomingMessage: find owner: %w", err)
	}

	status, err := s.GetStatus(ctx, owner.ID)
	if err != nil || !status.IsActive {
		return nil
	}

	settings, err := s.settingsRepo.FindByUserID(ctx, owner.ID)
	if errors.Is(err, repository.ErrNotFound) || !settings.AutoReplyEnabled {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deepWorkService.HandleIncomingMessage: find settings: %w", err)
	}

	if senderSlackID == ownerSlackID {
		return nil
	}

	if contains(settings.Blacklist, senderSlackID) {
		s.logger.Info("sender is blacklisted, skipping auto-reply",
			slog.String("sender", senderSlackID),
			slog.String("owner", ownerSlackID),
		)
		return nil
	}

	cooldownKey := cooldownKeyPrefix + owner.ID + ":" + senderSlackID
	exists, err := s.rdb.Exists(ctx, cooldownKey).Result()
	if err == nil && exists > 0 {
		s.logger.Info("sender in cooldown, skipping auto-reply",
			slog.String("sender", senderSlackID),
			slog.String("owner", ownerSlackID),
		)
		return nil
	}

	replyChannel := channelID
	replyThreadTS := ""
	if settings.ReplyInThread && threadTS != "" {
		replyThreadTS = threadTS
	} else if settings.ReplyInThread && originalTS != "" {
		replyThreadTS = originalTS
	}

	autoReplyText := settings.DefaultMessage
	sendErr := s.messenger.SendAutoReply(ctx, owner.AccessToken, replyChannel, autoReplyText, replyThreadTS)

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
		s.logger.Warn("failed to create message log", slog.String("error", logErr.Error()))
	}

	if sendErr != nil {
		return fmt.Errorf("deepWorkService.HandleIncomingMessage: send auto-reply: %w", sendErr)
	}

	cooldownDuration := time.Duration(settings.CooldownMinutes) * time.Minute
	if err := s.rdb.Set(ctx, cooldownKey, "1", cooldownDuration).Err(); err != nil {
		s.logger.Warn("failed to set cooldown in redis", slog.String("error", err.Error()))
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
