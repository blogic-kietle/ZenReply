// Package repository provides data access layer implementations for ZenReply.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kietle/zenreply/model"
)

// SessionRepository defines the contract for deep work session data access.
type SessionRepository interface {
	Create(ctx context.Context, userID, reason string) (*model.DeepWorkSession, error)
	FindActiveByUserID(ctx context.Context, userID string) (*model.DeepWorkSession, error)
	FindByID(ctx context.Context, id string) (*model.DeepWorkSession, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.DeepWorkSession, int64, error)
	End(ctx context.Context, id string) error
}

type sessionRepository struct {
	db *pgxpool.Pool
}

// NewSessionRepository creates a new SessionRepository backed by PostgreSQL.
func NewSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, userID, reason string) (*model.DeepWorkSession, error) {
	// Ensure no active session exists before creating a new one.
	const endPrev = `
		UPDATE deep_work_sessions SET is_active = FALSE, end_time = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND is_active = TRUE
	`
	if _, err := r.db.Exec(ctx, endPrev, userID); err != nil {
		return nil, fmt.Errorf("sessionRepository.Create: end previous: %w", err)
	}

	const q = `
		INSERT INTO deep_work_sessions (user_id, reason, start_time, is_active)
		VALUES ($1, $2, NOW(), TRUE)
		RETURNING id, user_id, reason, start_time, end_time, is_active, created_at, updated_at
	`
	s := &model.DeepWorkSession{}
	err := r.db.QueryRow(ctx, q, userID, reason).Scan(
		&s.ID, &s.UserID, &s.Reason, &s.StartTime, &s.EndTime, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("sessionRepository.Create: %w", err)
	}
	return s, nil
}

func (r *sessionRepository) FindActiveByUserID(ctx context.Context, userID string) (*model.DeepWorkSession, error) {
	const q = `
		SELECT id, user_id, reason, start_time, end_time, is_active, created_at, updated_at
		FROM deep_work_sessions WHERE user_id = $1 AND is_active = TRUE
		ORDER BY start_time DESC LIMIT 1
	`
	s := &model.DeepWorkSession{}
	err := r.db.QueryRow(ctx, q, userID).Scan(
		&s.ID, &s.UserID, &s.Reason, &s.StartTime, &s.EndTime, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sessionRepository.FindActiveByUserID: %w", err)
	}
	return s, nil
}

func (r *sessionRepository) FindByID(ctx context.Context, id string) (*model.DeepWorkSession, error) {
	const q = `
		SELECT id, user_id, reason, start_time, end_time, is_active, created_at, updated_at
		FROM deep_work_sessions WHERE id = $1
	`
	s := &model.DeepWorkSession{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.UserID, &s.Reason, &s.StartTime, &s.EndTime, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sessionRepository.FindByID: %w", err)
	}
	return s, nil
}

func (r *sessionRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.DeepWorkSession, int64, error) {
	const countQ = `SELECT COUNT(*) FROM deep_work_sessions WHERE user_id = $1`
	var total int64
	if err := r.db.QueryRow(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("sessionRepository.ListByUserID count: %w", err)
	}

	const q = `
		SELECT id, user_id, reason, start_time, end_time, is_active, created_at, updated_at
		FROM deep_work_sessions WHERE user_id = $1
		ORDER BY start_time DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("sessionRepository.ListByUserID: %w", err)
	}
	defer rows.Close()

	var sessions []*model.DeepWorkSession
	for rows.Next() {
		s := &model.DeepWorkSession{}
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Reason, &s.StartTime, &s.EndTime, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("sessionRepository.ListByUserID scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, total, nil
}

func (r *sessionRepository) End(ctx context.Context, id string) error {
	now := time.Now()
	const q = `
		UPDATE deep_work_sessions SET is_active = FALSE, end_time = $2, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
	`
	ct, err := r.db.Exec(ctx, q, id, now)
	if err != nil {
		return fmt.Errorf("sessionRepository.End: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
