package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/worker/internal/model"
)

type PostgresMessageStore struct {
	pool *pgxpool.Pool
}

func NewPostgresMessageStore(pool *pgxpool.Pool) *PostgresMessageStore {
	return &PostgresMessageStore{pool: pool}
}

func (s *PostgresMessageStore) Save(ctx context.Context, message model.Message) error {
	const upsertConversation = `
		INSERT INTO message.conversations AS c (id, member_a, member_b, last_message_id, last_message_at)
		VALUES ($1::uuid, LEAST($2::uuid, $3::uuid), GREATEST($2::uuid, $3::uuid), $4::uuid, $5)
		ON CONFLICT (id) DO UPDATE
		SET last_message_id = EXCLUDED.last_message_id,
		    last_message_at = EXCLUDED.last_message_at
		WHERE (EXCLUDED.last_message_at, EXCLUDED.last_message_id) > (c.last_message_at, c.last_message_id)`

	const insertMessage = `
		INSERT INTO message.messages (id, conversation_id, sender_id, recipient_id, body, client_id, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, upsertConversation,
			message.ConversationID, message.SenderID, message.RecipientID, message.ID, message.CreatedAt); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, insertMessage,
			message.ID, message.ConversationID, message.SenderID, message.RecipientID, message.Body, message.ClientID, message.CreatedAt)
		return err
	})
	return classify(err)
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code[:2] {
		case "22", "23":
			return fmt.Errorf("%w: %s (%s)", model.ErrPermanent, pgErr.Message, pgErr.Code)
		}
	}
	return fmt.Errorf("lưu tin nhắn: %w", err)
}
