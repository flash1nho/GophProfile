package repository

import (
	"context"
	"iter"
	"slices"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/flash1nho/GophProfile/pkg/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Ping(ctx context.Context) error
}

type AvatarRepository struct {
	db  DB
	cb  *gobreaker.CircuitBreaker
	log *zap.Logger
}

func NewAvatarRepository(db DB, log *zap.Logger) *AvatarRepository {
	return &AvatarRepository{
		db:  db,
		log: log,
		cb:  utils.NewCircuitBreaker("db"),
	}
}

func (r *AvatarRepository) Create(ctx context.Context, a *domain.Avatar) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "InsertAvatar")
	defer span.End()

	span.SetAttributes(attribute.String("operation", "insert"))

	_, err := r.cb.Execute(func() (any, error) {
		return nil, r.db.QueryRow(ctx, `
			INSERT INTO avatars (id, user_id, file_name, mime_type, size_bytes, s3_key)
			VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING created_at
		`,
			a.ID, a.UserID, a.FileName, a.MimeType, a.SizeBytes, a.S3Key,
		).Scan(&a.CreatedAt)
	})

	return err
}

func (r *AvatarRepository) GetAvatar(ctx context.Context, id string) (*domain.Avatar, error) {
	var a domain.Avatar

	_, err := r.cb.Execute(func() (any, error) {
		row := r.db.QueryRow(ctx, `
			SELECT id, user_id, file_name, mime_type, size_bytes, s3_key,
			       upload_status, processing_status, created_at, updated_at
			FROM avatars
			WHERE id=$1 AND deleted_at IS NULL
		`, id)

		return nil, row.Scan(
			&a.ID,
			&a.UserID,
			&a.FileName,
			&a.MimeType,
			&a.SizeBytes,
			&a.S3Key,
			&a.UploadStatus,
			&a.ProcessingStatus,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (r *AvatarRepository) SoftDelete(ctx context.Context, avatarID string) error {
	_, err := r.cb.Execute(func() (any, error) {
		return r.db.Exec(ctx, `
			UPDATE avatars
			SET deleted_at = NOW()
			WHERE id = $1
		`, avatarID)
	})

	return err
}

func (r *AvatarRepository) GetLatestByUser(ctx context.Context, userID string) (*domain.Avatar, error) {
	var a domain.Avatar

	_, err := r.cb.Execute(func() (any, error) {
		row := r.db.QueryRow(ctx, `
			SELECT id, user_id, file_name, mime_type, size_bytes, s3_key
			FROM avatars
			WHERE user_id=$1 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		`, userID)

		return nil, row.Scan(
			&a.ID,
			&a.UserID,
			&a.FileName,
			&a.MimeType,
			&a.SizeBytes,
			&a.S3Key,
		)
	})

	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (r *AvatarRepository) ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error) {
	var result []domain.Avatar

	_, _ = r.cb.Execute(func() (any, error) {
		for a := range r.listByUserIter(ctx, userID) {
			result = append(result, a)
		}
		return nil, nil
	})

	result = uniqueAvatars(result)

	return result, nil
}

func (r *AvatarRepository) UpdateUploadStatus(ctx context.Context, id string, status domain.UploadStatus) error {
	_, err := r.cb.Execute(func() (any, error) {
		return r.db.Exec(ctx, `
			UPDATE avatars
			SET upload_status = $1, updated_at = NOW()
			WHERE id = $2
		`, string(status), id)
	})

	return err
}

func (r *AvatarRepository) UpdateProcessingStatus(ctx context.Context, id string, status domain.ProcessingStatus) error {
	_, err := r.cb.Execute(func() (any, error) {
		return r.db.Exec(ctx, `
			UPDATE avatars
			SET processing_status = $1, updated_at = NOW()
			WHERE id = $2
		`, string(status), id)
	})

	return err
}

func (r *AvatarRepository) UpdateThumbnails(ctx context.Context, id string, thumbs map[string]string) error {
	_, err := r.cb.Execute(func() (any, error) {
		return r.db.Exec(ctx, `
			UPDATE avatars
			SET thumbnail_s3_keys = $1, updated_at = NOW()
			WHERE id = $2
		`, thumbs, id)
	})

	return err
}

func (r *AvatarRepository) Ping(ctx context.Context) error {
	_, err := r.cb.Execute(func() (any, error) {
		return nil, r.db.Ping(ctx)
	})

	return err
}

func (r *AvatarRepository) listByUserIter(ctx context.Context, userID string) iter.Seq[domain.Avatar] {
	return func(yield func(domain.Avatar) bool) {
		rows, err := r.db.Query(ctx, `
			SELECT id, user_id, file_name, mime_type, size_bytes,
			       s3_key, upload_status, processing_status,
			       created_at, updated_at
			FROM avatars
			WHERE user_id=$1 AND deleted_at IS NULL
			ORDER BY created_at DESC
		`, userID)

		if err != nil {
			return
		}
		defer rows.Close()

		for rows.Next() {
			var a domain.Avatar
			if err := rows.Scan(
				&a.ID,
				&a.UserID,
				&a.FileName,
				&a.MimeType,
				&a.SizeBytes,
				&a.S3Key,
				&a.UploadStatus,
				&a.ProcessingStatus,
				&a.CreatedAt,
				&a.UpdatedAt,
			); err != nil {
				return
			}

			if !yield(a) {
				return
			}
		}
	}
}

func uniqueAvatars(in []domain.Avatar) []domain.Avatar {
	return slices.CompactFunc(in, func(a, b domain.Avatar) bool {
		return a.ID == b.ID
	})
}
