// Package model defines the core domain entities for ZenReply.
package model

import (
	"time"
)

// User represents a ZenReply user authenticated via Slack OAuth.
type User struct {
	ID           string    `json:"id" db:"id"`
	SlackUserID  string    `json:"slack_user_id" db:"slack_user_id"`
	SlackTeamID  string    `json:"slack_team_id" db:"slack_team_id"`
	SlackName    string    `json:"slack_name" db:"slack_name"`
	Email        string    `json:"email" db:"email"`
	AvatarURL    string    `json:"avatar_url" db:"avatar_url"`
	// AccessToken is the Slack User OAuth token (encrypted at rest).
	AccessToken  string    `json:"-" db:"access_token"`
	// BotToken is the Slack Bot token for the workspace.
	BotToken     string    `json:"-" db:"bot_token"`
	TokenScope   string    `json:"token_scope" db:"token_scope"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserSettings holds per-user configuration for the auto-reply behavior.
type UserSettings struct {
	ID                  string    `json:"id" db:"id"`
	UserID              string    `json:"user_id" db:"user_id"`
	DefaultMessage      string    `json:"default_message" db:"default_message"`
	DefaultReason       string    `json:"default_reason" db:"default_reason"`
	// CooldownMinutes is the interval (in minutes) before re-sending an auto-reply to the same sender.
	CooldownMinutes     int       `json:"cooldown_minutes" db:"cooldown_minutes"`
	// Whitelist is a JSON array of Slack user IDs that always receive auto-replies.
	Whitelist           []string  `json:"whitelist" db:"whitelist"`
	// Blacklist is a JSON array of Slack user IDs that never receive auto-replies.
	Blacklist           []string  `json:"blacklist" db:"blacklist"`
	// ReplyInThread controls whether auto-replies are sent in the original message thread.
	ReplyInThread       bool      `json:"reply_in_thread" db:"reply_in_thread"`
	// NotifyOnResume controls whether to send a follow-up when the user ends deep work.
	NotifyOnResume      bool      `json:"notify_on_resume" db:"notify_on_resume"`
	AutoReplyEnabled    bool      `json:"auto_reply_enabled" db:"auto_reply_enabled"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// DeepWorkSession represents a single deep work session for a user.
type DeepWorkSession struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	Reason    string     `json:"reason" db:"reason"`
	StartTime time.Time  `json:"start_time" db:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" db:"end_time"`
	// IsActive is true when the session is currently ongoing.
	IsActive  bool       `json:"is_active" db:"is_active"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// MessageLog records every auto-reply sent by ZenReply.
type MessageLog struct {
	ID              string    `json:"id" db:"id"`
	UserID          string    `json:"user_id" db:"user_id"`
	SessionID       string    `json:"session_id" db:"session_id"`
	SenderSlackID   string    `json:"sender_slack_id" db:"sender_slack_id"`
	ChannelID       string    `json:"channel_id" db:"channel_id"`
	OriginalTS      string    `json:"original_ts" db:"original_ts"`
	ThreadTS        string    `json:"thread_ts" db:"thread_ts"`
	MessageText     string    `json:"message_text" db:"message_text"`
	AutoReplyText   string    `json:"auto_reply_text" db:"auto_reply_text"`
	WasSent         bool      `json:"was_sent" db:"was_sent"`
	ErrorMessage    string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// DeepWorkStatus is a lightweight view of a user's current deep work state (served from Redis).
type DeepWorkStatus struct {
	IsActive    bool       `json:"is_active"`
	SessionID   string     `json:"session_id,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	// TTL is the remaining seconds until the session auto-expires (if set).
	TTLSeconds  int64      `json:"ttl_seconds,omitempty"`
}
