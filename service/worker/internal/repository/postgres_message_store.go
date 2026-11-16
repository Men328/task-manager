package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

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
		INSERT INTO message.messages (id, conversation_id, sender_id, recipient_id, body, client_id, created_at,
		                              task_id, task_title, task_status_name, task_status_category, task_status_color, task_status_changed_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7,
		        NULLIF($8, '')::uuid, NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), $13)
		ON CONFLICT (id) DO NOTHING`

	var task model.TaskRef
	var changedAt *time.Time
	if message.Task != nil {
		task = *message.Task
		if !task.StatusChangedAt.IsZero() {
			changedAt = &task.StatusChangedAt
		}
	}

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, upsertConversation,
			message.ConversationID, message.SenderID, message.RecipientID, message.ID, message.CreatedAt); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, insertMessage,
			message.ID, message.ConversationID, message.SenderID, message.RecipientID, message.Body, message.ClientID, message.CreatedAt,
			task.TaskID, task.Title, task.StatusName, task.StatusCategory, task.StatusColor, changedAt)
		return err
	})
	return classify(err)
}

func (s *PostgresMessageStore) UpdateTaskStatus(ctx context.Context, event model.TaskStatusChanged) (model.Message, bool, error) {
	const query = `
		UPDATE message.messages
		SET task_status_name = $3, task_status_category = NULLIF($4, ''), task_status_color = NULLIF($5, ''), task_status_changed_at = $6
		WHERE id = $1::uuid AND task_id = $2::uuid
		  AND (task_status_changed_at IS NULL OR task_status_changed_at < $6)
		RETURNING id::text, conversation_id::text, sender_id::text, recipient_id::text, body, client_id, created_at,
		          task_id::text, COALESCE(task_title, ''), COALESCE(task_status_name, ''), COALESCE(task_status_category, ''),
		          COALESCE(task_status_color, ''), task_status_changed_at`

	var m model.Message
	var task model.TaskRef
	err := s.pool.QueryRow(ctx, query, event.MessageID, event.TaskID, event.ToStatusName, event.ToCategory, event.ToColor, event.ChangedAt).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.RecipientID, &m.Body, &m.ClientID, &m.CreatedAt,
			&task.TaskID, &task.Title, &task.StatusName, &task.StatusCategory, &task.StatusColor, &task.StatusChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Message{}, false, nil
	}
	if err != nil {
		return model.Message{}, false, classify(err)
	}
	m.Task = &task
	return m, true, nil
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
