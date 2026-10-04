package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/attachment/internal/model"
)

const attachmentColumns = `id::text, profile_id::text, owner_type, owner_id::text, file_name, content_type, size, object_key, created_at`

type PostgresAttachmentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAttachmentRepository(pool *pgxpool.Pool) *PostgresAttachmentRepository {
	return &PostgresAttachmentRepository{pool: pool}
}

func mapAttachmentError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503", "22P02", "23514":
			if strings.Contains(pgErr.ConstraintName, "owner_type") {
				return model.NewError(model.ErrorKindOwnerTypeInvalid, "unsupported owner type")
			}
			return model.ErrInvalid
		}
	}
	return err
}

func scanAttachment(row pgx.Row) (model.Attachment, error) {
	var item model.Attachment
	if err := row.Scan(
		&item.ID,
		&item.ProfileID,
		&item.OwnerType,
		&item.OwnerID,
		&item.FileName,
		&item.ContentType,
		&item.Size,
		&item.ObjectKey,
		&item.CreatedAt,
	); err != nil {
		return model.Attachment{}, mapAttachmentError(err)
	}
	return item, nil
}

func (r *PostgresAttachmentRepository) Create(ctx context.Context, item model.Attachment) (model.Attachment, error) {
	const query = `
		INSERT INTO attachment.attachments
			(id, profile_id, owner_type, owner_id, file_name, content_type, size, object_key)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3, $4::uuid,
			 $5, $6, $7, $8)
		RETURNING ` + attachmentColumns

	return scanAttachment(r.pool.QueryRow(ctx, query,
		item.ID, item.ProfileID, item.OwnerType, item.OwnerID,
		item.FileName, item.ContentType, item.Size, item.ObjectKey))
}

func (r *PostgresAttachmentRepository) Get(ctx context.Context, id string) (model.Attachment, error) {
	const query = `SELECT ` + attachmentColumns + ` FROM attachment.attachments WHERE id = $1::uuid`
	return scanAttachment(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresAttachmentRepository) List(ctx context.Context, f model.Filter) ([]model.Attachment, error) {
	const query = `
		SELECT ` + attachmentColumns + `
		FROM attachment.attachments
		WHERE ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2::text IS NULL OR owner_type = $2::text)
		  AND ($3::uuid IS NULL OR owner_id = $3::uuid)
		ORDER BY created_at DESC, id DESC`

	rows, err := r.pool.Query(ctx, query,
		nullableText(f.ProfileID),
		nullableText(f.OwnerType),
		nullableText(f.OwnerID),
	)
	if err != nil {
		return nil, mapAttachmentError(err)
	}
	defer rows.Close()

	out := make([]model.Attachment, 0)
	for rows.Next() {
		item, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, mapAttachmentError(rows.Err())
}

func (r *PostgresAttachmentRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM attachment.attachments WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapAttachmentError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
