package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, a *domain.Avatar) error
	GetAvatar(ctx context.Context, id string) (*domain.Avatar, error)
	SoftDelete(ctx context.Context, id string) error

	GetLatestByUser(ctx context.Context, userID string) (*domain.Avatar, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error)
	UpdateUploadStatus(ctx context.Context, id string, status domain.UploadStatus) error
	UpdateProcessingStatus(ctx context.Context, id string, status domain.ProcessingStatus) error
	UpdateThumbnails(ctx context.Context, id string, thumbs map[string]string) error

	Ping(ctx context.Context) error
}

type Storage interface {
	Upload(context.Context, string, []byte, string) error
	Download(context.Context, string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Health(ctx context.Context) error
}

type Publisher interface {
	Publish(ctx context.Context, event any) error
	Ping() error
}

type AvatarService struct {
	repo Repository
	s3   Storage
	pub  Publisher
	log  *zap.Logger
}

func NewAvatarService(r Repository, s Storage, p Publisher, l *zap.Logger) *AvatarService {
	return &AvatarService{repo: r, s3: s, pub: p, log: l}
}

func (s *AvatarService) Upload(
	ctx context.Context,
	userID, fileName, mime string,
	data []byte,
) (*domain.Avatar, error) {
	ctx, span := otel.Tracer("avatar-service").Start(ctx, "Upload")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
	)

	id := uuid.NewString()
	key := fmt.Sprintf("avatars/%s/original", id)

	avatar := &domain.Avatar{
		ID:               id,
		UserID:           userID,
		FileName:         fileName,
		MimeType:         mime,
		SizeBytes:        int64(len(data)),
		S3Key:            key,
		UploadStatus:     domain.UploadStatusUploading,
		ProcessingStatus: domain.ProcessingStatusPending,
	}

	s.log.Info("upload started", zap.String("user_id", userID))

	if err := s.repo.Create(ctx, avatar); err != nil {
		s.log.Error("create avatar failed",
			zap.String("avatar_id", id),
			zap.Error(err),
		)
		return nil, fmt.Errorf("create avatar: %w", err)
	}

	if err := s.s3.Upload(ctx, key, data, mime); err != nil {
		s.log.Error("s3 upload failed",
			zap.String("avatar_id", id),
			zap.Error(err),
		)

		if errStatus := s.repo.UpdateUploadStatus(ctx, id, domain.UploadStatusFailed); errStatus != nil {
			s.log.Error("failed to update upload status",
				zap.String("avatar_id", id),
				zap.Error(errStatus),
			)
		}

		if errProc := s.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed); errProc != nil {
			s.log.Error("failed to update processing status",
				zap.String("avatar_id", id),
				zap.Error(errProc),
			)
		}

		return nil, fmt.Errorf("upload to s3: %w", err)
	}

	if err := s.repo.UpdateUploadStatus(ctx, id, domain.UploadStatusUploaded); err != nil {
		s.log.Error("update upload status failed",
			zap.String("avatar_id", id),
			zap.Error(err),
		)

		if errDelete := s.s3.Delete(ctx, key); errDelete != nil {
			s.log.Error("failed to cleanup s3 after db error",
				zap.String("avatar_id", id),
				zap.Error(errDelete),
			)
		}

		if errStatus := s.repo.UpdateUploadStatus(ctx, id, domain.UploadStatusFailed); errStatus != nil {
			s.log.Error("failed to mark upload as failed",
				zap.String("avatar_id", id),
				zap.Error(errStatus),
			)
		}

		if errProc := s.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed); errProc != nil {
			s.log.Error("failed to mark processing as failed",
				zap.String("avatar_id", id),
				zap.Error(errProc),
			)
		}

		return nil, fmt.Errorf("update upload status: %w", err)
	}

	avatar.UploadStatus = domain.UploadStatusUploaded

	if err := s.pub.Publish(ctx, map[string]string{
		"avatar_id": id,
		"user_id":   userID,
		"s3_key":    key,
	}); err != nil {
		s.log.Error("publish failed",
			zap.String("avatar_id", id),
			zap.Error(err),
		)

		if errProc := s.repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusFailed); errProc != nil {
			s.log.Error("failed to update processing status after publish error",
				zap.String("avatar_id", id),
				zap.Error(errProc),
			)
		}

		return avatar, fmt.Errorf("publish event: %w", err)
	}

	return avatar, nil
}

func (s *AvatarService) Get(ctx context.Context, id, size string) ([]byte, string, error) {
	avatar, err := s.repo.GetAvatar(ctx, id)
	if err != nil {
		return nil, "", err
	}

	var key string

	switch size {
	case "100x100", "300x300":
		if avatar.ThumbnailKeys != nil {
			if k, ok := avatar.ThumbnailKeys[size]; ok {
				key = k
			}
		}
	}

	if key == "" {
		key = avatar.S3Key
	}

	data, err := s.s3.Download(ctx, key)
	if err != nil {
		return nil, "", err
	}

	return data, avatar.MimeType, nil
}

func (s *AvatarService) GetByUser(ctx context.Context, userID string) (*domain.Avatar, error) {
	return s.repo.GetLatestByUser(ctx, userID)
}

func (s *AvatarService) Delete(ctx context.Context, id, userID string) error {
	avatar, err := s.repo.GetAvatar(ctx, id)
	if err != nil {
		return err
	}

	if avatar == nil {
		return fmt.Errorf("avatar not found")
	}

	if avatar.UserID != userID {
		return fmt.Errorf("forbidden")
	}

	return s.repo.SoftDelete(ctx, id)
}

func (s *AvatarService) DeleteByUser(ctx context.Context, userID string) error {
	avatar, err := s.repo.GetLatestByUser(ctx, userID)
	if err != nil {
		return err
	}

	if avatar == nil {
		return fmt.Errorf("avatar not found")
	}

	return s.Delete(ctx, avatar.ID, userID)
}

func (s *AvatarService) ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *AvatarService) GetMetadata(ctx context.Context, id string) (*domain.Avatar, error) {
	return s.repo.GetAvatar(ctx, id)
}

func (s *AvatarService) PingDB(ctx context.Context) error {
	if s.repo == nil {
		return fmt.Errorf("repo not initialized")
	}
	return s.repo.Ping(ctx)
}

func (s *AvatarService) PingS3(ctx context.Context) error {
	if s.s3 == nil {
		return fmt.Errorf("s3 not initialized")
	}
	return s.s3.Health(ctx)
}

func (s *AvatarService) PingRabbit(ctx context.Context) error {
	if s.pub == nil {
		return fmt.Errorf("publisher not initialized")
	}
	return s.pub.Ping()
}
