package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flash1nho/GophProfile/internal/repository"
	"github.com/flash1nho/GophProfile/pkg/storage"
	"github.com/flash1nho/GophProfile/pkg/utils"
)

type Worker struct {
	repo *repository.AvatarRepository
	s3   *storage.S3
}

func NewWorker(repo *repository.AvatarRepository, s3 *storage.S3) *Worker {
	return &Worker{
		repo: repo,
		s3:   s3,
	}
}

type UploadEvent struct {
	AvatarID string `json:"avatar_id"`
	S3Key    string `json:"s3_key"`
}

func (w *Worker) Handle(message []byte) error {
	var event UploadEvent

	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	ctx := context.Background()

	img, err := w.s3.Download(ctx, event.S3Key)
	if err != nil {
		return err
	}

	sizes := map[string][2]int{
		"100x100": {100, 100},
		"300x300": {300, 300},
	}

	result := make(map[string]string)

	for size, dim := range sizes {
		resized, err := utils.Resize(img, dim[0], dim[1])
		if err != nil {
			return err
		}

		key := fmt.Sprintf("thumbnails/%s/%s.jpg", event.AvatarID, size)

		if err := w.s3.Upload(ctx, key, resized, "image/jpeg"); err != nil {
			return err
		}

		result[size] = key
	}

	err = w.repo.UpdateThumbnails(ctx, event.AvatarID, result)
	if err != nil {
		return err
	}

	return w.repo.UpdateProcessingStatus(ctx, event.AvatarID, "completed")
}
