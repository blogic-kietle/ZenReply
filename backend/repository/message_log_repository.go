// Package repository provides data access layer implementations for ZenReply.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kietle/zenreply/model"
)

// MessageLogRepository defines the contract for message log data access.
type MessageLogRepository interface {
	Create(ctx context.Context, log *model.MessageLog) (*model.MessageLog, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.MessageLog, int64, error)
	ListBySessionID(ctx context.Context, sessionID string) ([]*model.MessageLog, error)
}

type messageLogRepository struct {
	db *pgxpool.Pool
}

// NewMessageLogRepository creates a new MessageLogRepository backed by PostgreSQL.
func NewMessageLogRepository(db *pgxpool.Pool) MessageLogRepository {
	return &messageLogRepository{db: db}
}

func (r *messageLogRepository) Create(ctx context.Context, l *model.MessageLog) (*model.MessageLog, error) {
	const q = `
		INSERT INTO message_logs
		    (user_id, session_id, sender_slack_id, channel_id, original_ts, thread_ts,
		     message_text, auto_reply_text, was_sent, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, user_id, session_id, sender_slack_id, channel_id, original_ts, thread_ts,
		          message_text, auto_reply_text, was_sent, error_message, created_at
	`
	result := &model.MessageLog{}
	err := r.db.QueryRow(ctx, q,
		l.UserID, l.SessionID, l.SenderSlackID, l.ChannelID,
		l.OriginalTS, l.ThreadTS, l.MessageText, l.AutoReplyText,
		l.WasSent, l.ErrorMessage,
	).Scan(
		&result.ID, &result.UserID, &result.SessionID, &result.SenderSlackID,
		&result.ChannelID, &result.OriginalTS, &result.ThreadTS,
		&result.MessageText, &result.AutoReplyText, &result.WasSent,
		&result.ErrorMessage, &result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("messageLogRepository.Create: %w", err)
	}
	return result, nil
}

func (r *messageLogRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.MessageLog, int64, error) {
	const countQ = `SELECT COUNT(*) FROM message_logs WHERE user_id = $1`
	var total int64
	if err := r.db.QueryRow(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("messageLogRepository.ListByUserID count: %w", err)
	}

	const q = `
		SELECT id, user_id, session_id, sender_slack_id, channel_id, original_ts, thread_ts,
		       message_text, auto_reply_text, was_sent, error_message, created_at
		FROM message_logs WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("messageLogRepository.ListByUserID: %w", err)
	}
	defer rows.Close()

	var logs []*model.MessageLog
	for rows.Next() {
		l := &model.MessageLog{}
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.SessionID, &l.SenderSlackID, &l.ChannelID,
			&l.OriginalTS, &l.ThreadTS, &l.MessageText, &l.AutoReplyText,
			&l.WasSent, &l.ErrorMessage, &l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("messageLogRepository.ListByUserID scan: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (r *messageLogRepository) ListBySessionID(ctx context.Context, sessionID string) ([]*model.MessageLog, error) {
	const q = `
		SELECT id, user_id, session_id, sender_slack_id, channel_id, original_ts, thread_ts,
		       message_text, auto_reply_text, was_sent, error_message, created_at
		FROM message_logs WHERE session_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("messageLogRepository.ListBySessionID: %w", err)
	}
	defer rows.Close()

	var logs []*model.MessageLog
	for rows.Next() {
		l := &model.MessageLog{}
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.SessionID, &l.SenderSlackID, &l.ChannelID,
			&l.OriginalTS, &l.ThreadTS, &l.MessageText, &l.AutoReplyText,
			&l.WasSent, &l.ErrorMessage, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("messageLogRepository.ListBySessionID scan: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}
