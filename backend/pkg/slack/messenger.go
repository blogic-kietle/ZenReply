// Package slack provides Slack API integration utilities for ZenReply.
package slack

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	slacklib "github.com/slack-go/slack"
)

const (
	maxRetries     = 3
	baseRetryDelay = 500 * time.Millisecond
)

// Messenger sends messages via the Slack API with retry logic.
type Messenger struct {
	logger *slog.Logger
}

// NewMessenger creates a new Messenger.
func NewMessenger(logger *slog.Logger) *Messenger {
	return &Messenger{logger: logger}
}

// SendAutoReply sends an auto-reply message using the user's personal access token.
// If threadTS is non-empty, the reply is sent within the original thread.
func (m *Messenger) SendAutoReply(ctx context.Context, userToken, channel, text, threadTS string) error {
	client := slacklib.New(userToken)

	opts := []slacklib.MsgOption{
		slacklib.MsgOptionText(text, false),
		slacklib.MsgOptionAsUser(true),
	}

	if threadTS != "" {
		opts = append(opts, slacklib.MsgOptionTS(threadTS))
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * baseRetryDelay
			m.logger.Warn("retrying slack message send",
				slog.Int("attempt", attempt),
				slog.Duration("delay", delay),
				slog.String("channel", channel),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		_, _, err := client.PostMessageContext(ctx, channel, opts...)
		if err == nil {
			return nil
		}

		lastErr = err
		m.logger.Error("failed to send slack message",
			slog.Int("attempt", attempt+1),
			slog.String("error", err.Error()),
			slog.String("channel", channel),
		)

		// Do not retry on non-retryable errors.
		if isNonRetryable(err) {
			break
		}
	}

	return fmt.Errorf("failed to send slack message after %d attempts: %w", maxRetries+1, lastErr)
}

// SendDM sends a direct message to a Slack user by their user ID.
func (m *Messenger) SendDM(ctx context.Context, userToken, recipientUserID, text string) error {
	return m.SendAutoReply(ctx, userToken, recipientUserID, text, "")
}

func isNonRetryable(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	nonRetryable := []string{
		"channel_not_found",
		"not_in_channel",
		"invalid_auth",
		"account_inactive",
		"token_revoked",
		"no_permission",
		"missing_scope",
	}
	for _, code := range nonRetryable {
		if errStr == code {
			return true
		}
	}
	return false
}
