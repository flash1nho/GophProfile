package services

import (
	"context"
	"fmt"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/google/uuid"
)

type Repository interface {
	Create(context.Context, *domain.Avatar) error
	GetByID(context.Context, string) (*domain.Avatar, error)
	SoftDelete(context.Context, string) error
}

type Storage interface {
	Upload(context.Context, string, []byte, string) error
	Download(context.Context, string) ([]byte, error)
}

type Publisher interface {
	Publish(any) error
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
		// логируем, но не валим основной поток
		// (важный прод-паттерн)
		fmt.Printf("publish error: %v\n", err)
	}

	return avatar, nil
}
