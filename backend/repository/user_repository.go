// Package repository provides data access layer implementations for ZenReply.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kietle/zenreply/model"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("record not found")

// UserRepository defines the contract for user data access.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindBySlackUserID(ctx context.Context, slackUserID string) (*model.User, error)
	Upsert(ctx context.Context, user *model.User) (*model.User, error)
	UpdateTokens(ctx context.Context, id, accessToken, botToken, tokenScope string) error
	Deactivate(ctx context.Context, id string) error
}

type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository backed by PostgreSQL.
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	const q = `
		SELECT id, slack_user_id, slack_team_id, slack_name, email, avatar_url,
		       access_token, bot_token, token_scope, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.SlackUserID, &u.SlackTeamID, &u.SlackName, &u.Email, &u.AvatarURL,
		&u.AccessToken, &u.BotToken, &u.TokenScope, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}
	return u, nil
}

func (r *userRepository) FindBySlackUserID(ctx context.Context, slackUserID string) (*model.User, error) {
	const q = `
		SELECT id, slack_user_id, slack_team_id, slack_name, email, avatar_url,
		       access_token, bot_token, token_scope, is_active, created_at, updated_at
		FROM users WHERE slack_user_id = $1
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, q, slackUserID).Scan(
		&u.ID, &u.SlackUserID, &u.SlackTeamID, &u.SlackName, &u.Email, &u.AvatarURL,
		&u.AccessToken, &u.BotToken, &u.TokenScope, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepository.FindBySlackUserID: %w", err)
	}
	return u, nil
}

// Upsert inserts a new user or updates existing fields on conflict.
func (r *userRepository) Upsert(ctx context.Context, user *model.User) (*model.User, error) {
	const q = `
		INSERT INTO users (slack_user_id, slack_team_id, slack_name, email, avatar_url, access_token, bot_token, token_scope, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE)
		ON CONFLICT (slack_user_id) DO UPDATE SET
			slack_team_id = EXCLUDED.slack_team_id,
			slack_name    = EXCLUDED.slack_name,
			email         = EXCLUDED.email,
			avatar_url    = EXCLUDED.avatar_url,
			access_token  = EXCLUDED.access_token,
			bot_token     = EXCLUDED.bot_token,
			token_scope   = EXCLUDED.token_scope,
			is_active     = TRUE,
			updated_at    = NOW()
		RETURNING id, slack_user_id, slack_team_id, slack_name, email, avatar_url,
		          access_token, bot_token, token_scope, is_active, created_at, updated_at
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, q,
		user.SlackUserID, user.SlackTeamID, user.SlackName, user.Email, user.AvatarURL,
		user.AccessToken, user.BotToken, user.TokenScope,
	).Scan(
		&u.ID, &u.SlackUserID, &u.SlackTeamID, &u.SlackName, &u.Email, &u.AvatarURL,
		&u.AccessToken, &u.BotToken, &u.TokenScope, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("userRepository.Upsert: %w", err)
	}
	return u, nil
}

func (r *userRepository) UpdateTokens(ctx context.Context, id, accessToken, botToken, tokenScope string) error {
	const q = `
		UPDATE users SET access_token = $2, bot_token = $3, token_scope = $4, updated_at = NOW()
		WHERE id = $1
	`
	ct, err := r.db.Exec(ctx, q, id, accessToken, botToken, tokenScope)
	if err != nil {
		return fmt.Errorf("userRepository.UpdateTokens: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) Deactivate(ctx context.Context, id string) error {
	const q = `UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("userRepository.Deactivate: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
