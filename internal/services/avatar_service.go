package services

import (
	"context"
	"fmt"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, a *domain.Avatar) error
	GetByID(ctx context.Context, id string) (*domain.Avatar, error)
	SoftDelete(ctx context.Context, id string) error

	GetLatestByUser(ctx context.Context, userID string) (*domain.Avatar, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error)
	UpdateThumbnails(ctx context.Context, id string, thumbs map[string]string) error

	Ping(ctx context.Context) error
}

type Storage interface {
	Upload(context.Context, string, []byte, string) error
	Download(context.Context, string) ([]byte, error)
	Health(ctx context.Context) error
}

type Publisher interface {
	Publish(any) error
	Ping() error
}

type AvatarService struct {
	repo Repository
	s3   Storage
	pub  Publisher
}

func NewAvatarService(r Repository, s Storage, p Publisher) *AvatarService {
	return &AvatarService{repo: r, s3: s, pub: p}
}

func (s *AvatarService) Upload(ctx context.Context, userID, fileName, mime string, data []byte) (*domain.Avatar, error) {
	id := uuid.NewString()
	key := fmt.Sprintf("avatars/%s/original", id)

	if err := s.s3.Upload(ctx, key, data, mime); err != nil {
		return nil, err
	}

	avatar := &domain.Avatar{
		ID:               id,
		UserID:           userID,
		FileName:         fileName,
		MimeType:         mime,
		SizeBytes:        int64(len(data)),
		S3Key:            key,
		UploadStatus:     "done",
		ProcessingStatus: "pending",
	}

	if err := s.repo.Create(ctx, avatar); err != nil {
		return nil, err
	}

	if err := s.pub.Publish(map[string]string{
		"avatar_id": id,
		"user_id":   userID,
		"s3_key":    key,
	}); err != nil {
		fmt.Printf("publish error: %v\n", err)
	}

	return avatar, nil
}

func (s *AvatarService) Get(ctx context.Context, id, size string) ([]byte, string, error) {
	avatar, err := s.repo.GetByID(ctx, id)
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
	avatar, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if avatar.UserID != userID {
		return fmt.Errorf("forbidden")
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}

	keys := []string{avatar.S3Key}

	for _, k := range avatar.ThumbnailKeys {
		keys = append(keys, k)
	}

	if err := s.pub.Publish(map[string]any{
		"avatar_id": id,
		"s3_keys":   keys,
	}); err != nil {
		fmt.Printf("publish delete event error: %v\n", err)
	}

	return nil
}

func (s *AvatarService) DeleteByUser(ctx context.Context, userID string) error {
	avatar, err := s.repo.GetLatestByUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.Delete(ctx, avatar.ID, userID)
}

func (s *AvatarService) ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *AvatarService) GetMetadata(ctx context.Context, id string) (*domain.Avatar, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AvatarService) PingDB(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *AvatarService) PingS3(ctx context.Context) error {
	return s.s3.Health(ctx)
}

func (s *AvatarService) PingRabbit(ctx context.Context) error {
	return s.pub.Ping()
}
