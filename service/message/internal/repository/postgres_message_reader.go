package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/message/internal/model"
)

type PostgresMessageReader struct {
	pool *pgxpool.Pool
}

func NewPostgresMessageReader(pool *pgxpool.Pool) *PostgresMessageReader {
	return &PostgresMessageReader{pool: pool}
}

func (r *PostgresMessageReader) ListMessages(ctx context.Context, conversationID string, before time.Time, limit int) ([]model.Message, error) {
	const query = `
		SELECT id::text, conversation_id::text, sender_id::text, recipient_id::text, body, client_id, created_at
		FROM message.messages
		WHERE conversation_id = $1::uuid
		  AND ($2::timestamptz IS NULL OR created_at < $2::timestamptz)
		ORDER BY created_at DESC, id DESC
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
		var m model.Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.RecipientID, &m.Body, &m.ClientID, &m.CreatedAt)
		return m, err
	})
}

func (r *PostgresMessageReader) ListConversations(ctx context.Context, profileID string, limit int) ([]model.Conversation, error) {
	const query = `
		SELECT c.id::text,
		       CASE WHEN c.member_a = $1::uuid THEN c.member_b::text ELSE c.member_a::text END,
		       m.id::text, m.conversation_id::text, m.sender_id::text, m.recipient_id::text, m.body, m.client_id, m.created_at
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
		m := &c.LastMessage
		err := row.Scan(&c.ID, &c.PeerID, &m.ID, &m.ConversationID, &m.SenderID, &m.RecipientID, &m.Body, &m.ClientID, &m.CreatedAt)
		return c, err
	})
}

type EmptyMessageReader struct{}

func (EmptyMessageReader) ListMessages(context.Context, string, time.Time, int) ([]model.Message, error) {
	return []model.Message{}, nil
}

func (EmptyMessageReader) ListConversations(context.Context, string, int) ([]model.Conversation, error) {
	return []model.Conversation{}, nil
}
