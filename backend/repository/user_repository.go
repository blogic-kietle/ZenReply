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
	UpdateToken(ctx context.Context, id, accessToken, tokenScope string) error
	Deactivate(ctx context.Context, id string) error
}

type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository backed by PostgreSQL.
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

const userSelectCols = `id, slack_user_id, slack_team_id, slack_name, email, avatar_url,
                        access_token, token_scope, is_active, created_at, updated_at`

func scanUser(row pgx.Row) (*model.User, error) {
	u := &model.User{}
	err := row.Scan(
		&u.ID, &u.SlackUserID, &u.SlackTeamID, &u.SlackName, &u.Email, &u.AvatarURL,
		&u.AccessToken, &u.TokenScope, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	q := `SELECT ` + userSelectCols + ` FROM users WHERE id = $1`
	u, err := scanUser(r.db.QueryRow(ctx, q, id))
	if err != nil {
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}
	return u, nil
}

func (r *userRepository) FindBySlackUserID(ctx context.Context, slackUserID string) (*model.User, error) {
	q := `SELECT ` + userSelectCols + ` FROM users WHERE slack_user_id = $1`
	u, err := scanUser(r.db.QueryRow(ctx, q, slackUserID))
	if err != nil {
		return nil, fmt.Errorf("userRepository.FindBySlackUserID: %w", err)
	}
	return u, nil
}

// Upsert inserts a new user or refreshes their token and profile on conflict.
func (r *userRepository) Upsert(ctx context.Context, user *model.User) (*model.User, error) {
	q := `
		INSERT INTO users (slack_user_id, slack_team_id, slack_name, email, avatar_url, access_token, token_scope, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
		ON CONFLICT (slack_user_id) DO UPDATE SET
			slack_team_id = EXCLUDED.slack_team_id,
			slack_name    = EXCLUDED.slack_name,
			email         = EXCLUDED.email,
			avatar_url    = EXCLUDED.avatar_url,
			access_token  = EXCLUDED.access_token,
			token_scope   = EXCLUDED.token_scope,
			is_active     = TRUE,
			updated_at    = NOW()
		RETURNING ` + userSelectCols
	u, err := scanUser(r.db.QueryRow(ctx, q,
		user.SlackUserID, user.SlackTeamID, user.SlackName, user.Email, user.AvatarURL,
		user.AccessToken, user.TokenScope,
	))
	if err != nil {
		return nil, fmt.Errorf("userRepository.Upsert: %w", err)
	}
	return u, nil
}

// UpdateToken refreshes the user's Slack access token (e.g., after token rotation).
func (r *userRepository) UpdateToken(ctx context.Context, id, accessToken, tokenScope string) error {
	q := `UPDATE users SET access_token = $2, token_scope = $3, updated_at = NOW() WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id, accessToken, tokenScope)
	if err != nil {
		return fmt.Errorf("userRepository.UpdateToken: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) Deactivate(ctx context.Context, id string) error {
	q := `UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("userRepository.Deactivate: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
