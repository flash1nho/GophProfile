package repository

import (
	"context"
	"encoding/json"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AvatarRepository struct {
	db *pgxpool.Pool
}

func NewAvatarRepository(db *pgxpool.Pool) *AvatarRepository {
	return &AvatarRepository{db: db}
}

func (r *AvatarRepository) Create(ctx context.Context, a *domain.Avatar) error {
	_, err := r.db.Exec(ctx, `
	INSERT INTO avatars (id, user_id, file_name, mime_type, size_bytes, s3_key, upload_status, processing_status)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`,
		a.ID, a.UserID, a.FileName, a.MimeType, a.SizeBytes, a.S3Key,
		a.UploadStatus, a.ProcessingStatus,
	)
	return err
}

func (r *AvatarRepository) GetByID(ctx context.Context, id string) (*domain.Avatar, error) {
	row := r.db.QueryRow(ctx, `
	SELECT id, user_id, file_name, mime_type, size_bytes, s3_key
	FROM avatars WHERE id=$1 AND deleted_at IS NULL
	`, id)

	var a domain.Avatar
	if err := row.Scan(&a.ID, &a.UserID, &a.FileName, &a.MimeType, &a.SizeBytes, &a.S3Key); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AvatarRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE avatars SET deleted_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *AvatarRepository) GetLatestByUser(ctx context.Context, userID string) (*domain.Avatar, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, file_name, mime_type, size_bytes, s3_key
		 FROM avatars WHERE user_id=$1 AND deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT 1`,
		userID,
	)

	var a domain.Avatar
	if err := row.Scan(&a.ID, &a.UserID, &a.FileName, &a.MimeType, &a.SizeBytes, &a.S3Key); err != nil {
		return nil, err
	}

	return &a, nil
}

func (r *AvatarRepository) ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, file_name FROM avatars WHERE user_id=$1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Avatar

	for rows.Next() {
		var a domain.Avatar
		if err := rows.Scan(&a.ID, &a.FileName); err != nil {
			return nil, err
		}
		list = append(list, a)
	}

	return list, nil
}

func (r *AvatarRepository) UpdateThumbnails(ctx context.Context, id string, thumbs map[string]string) error {
	data, err := json.Marshal(thumbs)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`UPDATE avatars
		 SET thumbnail_s3_keys = $1,
		     processing_status = 'completed',
		     updated_at = NOW()
		 WHERE id = $2`,
		data,
		id,
	)

	return err
}

func (r *AvatarRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}
