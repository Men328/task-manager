package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/message/internal/model"
)

const messageColumns = `m.id::text, m.conversation_id::text, m.sender_id::text, m.recipient_id::text, m.body, m.client_id, m.created_at,
	COALESCE(m.task_id::text, ''), COALESCE(m.task_title, ''), COALESCE(m.task_status_name, ''),
	COALESCE(m.task_status_category, ''), COALESCE(m.task_status_color, ''), m.task_status_changed_at`

func scanMessage(row pgx.Row, prefix ...any) (model.Message, error) {
	var m model.Message
	var task model.TaskRef
	var changedAt *time.Time
	targets := append(prefix, &m.ID, &m.ConversationID, &m.SenderID, &m.RecipientID, &m.Body, &m.ClientID, &m.CreatedAt,
		&task.TaskID, &task.Title, &task.StatusName, &task.StatusCategory, &task.StatusColor, &changedAt)
	if err := row.Scan(targets...); err != nil {
		return model.Message{}, err
	}
	if task.TaskID != "" {
		if changedAt != nil {
			task.StatusChangedAt = *changedAt
		}
		m.Task = &task
	}
	return m, nil
}

type PostgresMessageReader struct {
	pool *pgxpool.Pool
}

func NewPostgresMessageReader(pool *pgxpool.Pool) *PostgresMessageReader {
	return &PostgresMessageReader{pool: pool}
}

func (r *PostgresMessageReader) ListMessages(ctx context.Context, conversationID string, before time.Time, limit int) ([]model.Message, error) {
	const query = `
		SELECT ` + messageColumns + `
		FROM message.messages m
		WHERE m.conversation_id = $1::uuid
		  AND ($2::timestamptz IS NULL OR m.created_at < $2::timestamptz)
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $3`

	var cursor *time.Time
	if !before.IsZero() {
		cursor = &before
	}
	rows, err := r.pool.Query(ctx, query, conversationID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("đọc tin nhắn: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Message, error) {
		return scanMessage(row)
	})
}

func (r *PostgresMessageReader) ListConversations(ctx context.Context, profileID string, limit int) ([]model.Conversation, error) {
	const query = `
		SELECT c.id::text,
		       CASE WHEN c.member_a = $1::uuid THEN c.member_b::text ELSE c.member_a::text END,
		       ` + messageColumns + `
		FROM message.conversations c
		JOIN message.messages m ON m.id = c.last_message_id
		WHERE c.member_a = $1::uuid OR c.member_b = $1::uuid
		ORDER BY c.last_message_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, profileID, limit)
	if err != nil {
		return nil, fmt.Errorf("đọc cuộc trò chuyện: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Conversation, error) {
		var c model.Conversation
		last, err := scanMessage(row, &c.ID, &c.PeerID)
		c.LastMessage = last
		return c, err
	})
}

func (r *PostgresMessageReader) GetMessage(ctx context.Context, id string) (model.Message, error) {
	const query = `SELECT ` + messageColumns + ` FROM message.messages m WHERE m.id = $1::uuid`
	message, err := scanMessage(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Message{}, model.ErrNotFound
	}
	if err != nil {
		return model.Message{}, fmt.Errorf("đọc tin nhắn: %w", err)
	}
	return message, nil
}

type EmptyMessageReader struct{}

func (EmptyMessageReader) GetMessage(context.Context, string) (model.Message, error) {
	return model.Message{}, model.ErrNotFound
}

func (EmptyMessageReader) ListMessages(context.Context, string, time.Time, int) ([]model.Message, error) {
	return []model.Message{}, nil
}

func (EmptyMessageReader) ListConversations(context.Context, string, int) ([]model.Conversation, error) {
	return []model.Conversation{}, nil
}
