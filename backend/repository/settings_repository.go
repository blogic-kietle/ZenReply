// Package repository provides data access layer implementations for ZenReply.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kietle/zenreply/model"
)

// SettingsRepository defines the contract for user settings data access.
type SettingsRepository interface {
	FindByUserID(ctx context.Context, userID string) (*model.UserSettings, error)
	Upsert(ctx context.Context, settings *model.UserSettings) (*model.UserSettings, error)
	UpdateWhitelist(ctx context.Context, userID string, whitelist []string) error
	UpdateBlacklist(ctx context.Context, userID string, blacklist []string) error
}

type settingsRepository struct {
	db *pgxpool.Pool
}

// NewSettingsRepository creates a new SettingsRepository backed by PostgreSQL.
func NewSettingsRepository(db *pgxpool.Pool) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) FindByUserID(ctx context.Context, userID string) (*model.UserSettings, error) {
	const q = `
		SELECT id, user_id, default_message, default_reason, cooldown_minutes,
		       whitelist, blacklist, reply_in_thread, notify_on_resume, auto_reply_enabled,
		       created_at, updated_at
		FROM user_settings WHERE user_id = $1
	`
	s := &model.UserSettings{}
	var whitelistJSON, blacklistJSON []byte

	err := r.db.QueryRow(ctx, q, userID).Scan(
		&s.ID, &s.UserID, &s.DefaultMessage, &s.DefaultReason, &s.CooldownMinutes,
		&whitelistJSON, &blacklistJSON,
		&s.ReplyInThread, &s.NotifyOnResume, &s.AutoReplyEnabled,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("settingsRepository.FindByUserID: %w", err)
	}

	if err := json.Unmarshal(whitelistJSON, &s.Whitelist); err != nil {
		s.Whitelist = []string{}
	}
	if err := json.Unmarshal(blacklistJSON, &s.Blacklist); err != nil {
		s.Blacklist = []string{}
	}

	return s, nil
}

// Upsert inserts or updates user settings, creating defaults on first use.
func (r *settingsRepository) Upsert(ctx context.Context, s *model.UserSettings) (*model.UserSettings, error) {
	whitelistJSON, err := json.Marshal(s.Whitelist)
	if err != nil {
		return nil, fmt.Errorf("settingsRepository.Upsert: marshal whitelist: %w", err)
	}
	blacklistJSON, err := json.Marshal(s.Blacklist)
	if err != nil {
		return nil, fmt.Errorf("settingsRepository.Upsert: marshal blacklist: %w", err)
	}

	const q = `
		INSERT INTO user_settings (user_id, default_message, default_reason, cooldown_minutes,
		                           whitelist, blacklist, reply_in_thread, notify_on_resume, auto_reply_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			default_message    = EXCLUDED.default_message,
			default_reason     = EXCLUDED.default_reason,
			cooldown_minutes   = EXCLUDED.cooldown_minutes,
			whitelist          = EXCLUDED.whitelist,
			blacklist          = EXCLUDED.blacklist,
			reply_in_thread    = EXCLUDED.reply_in_thread,
			notify_on_resume   = EXCLUDED.notify_on_resume,
			auto_reply_enabled = EXCLUDED.auto_reply_enabled,
			updated_at         = NOW()
		RETURNING id, user_id, default_message, default_reason, cooldown_minutes,
		          whitelist, blacklist, reply_in_thread, notify_on_resume, auto_reply_enabled,
		          created_at, updated_at
	`
	result := &model.UserSettings{}
	var wl, bl []byte

	err = r.db.QueryRow(ctx, q,
		s.UserID, s.DefaultMessage, s.DefaultReason, s.CooldownMinutes,
		whitelistJSON, blacklistJSON,
		s.ReplyInThread, s.NotifyOnResume, s.AutoReplyEnabled,
	).Scan(
		&result.ID, &result.UserID, &result.DefaultMessage, &result.DefaultReason, &result.CooldownMinutes,
		&wl, &bl,
		&result.ReplyInThread, &result.NotifyOnResume, &result.AutoReplyEnabled,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("settingsRepository.Upsert: %w", err)
	}

	_ = json.Unmarshal(wl, &result.Whitelist)
	_ = json.Unmarshal(bl, &result.Blacklist)

	return result, nil
}

func (r *settingsRepository) UpdateWhitelist(ctx context.Context, userID string, whitelist []string) error {
	data, err := json.Marshal(whitelist)
	if err != nil {
		return fmt.Errorf("settingsRepository.UpdateWhitelist: %w", err)
	}
	const q = `UPDATE user_settings SET whitelist = $2, updated_at = NOW() WHERE user_id = $1`
	_, err = r.db.Exec(ctx, q, userID, data)
	return err
}

func (r *settingsRepository) UpdateBlacklist(ctx context.Context, userID string, blacklist []string) error {
	data, err := json.Marshal(blacklist)
	if err != nil {
		return fmt.Errorf("settingsRepository.UpdateBlacklist: %w", err)
	}
	const q = `UPDATE user_settings SET blacklist = $2, updated_at = NOW() WHERE user_id = $1`
	_, err = r.db.Exec(ctx, q, userID, data)
	return err
}
