package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

type DeleteEvent struct {
	AvatarID string   `json:"avatar_id"`
	S3Keys   []string `json:"s3_keys"`
}

func (w *Worker) HandleUploadEvent(message []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var upload UploadEvent
	if err := json.Unmarshal(message, &upload); err == nil && upload.AvatarID != "" && upload.S3Key != "" {
		return retry(ctx, 5, time.Second, func() error {
			return w.handleUpload(ctx, upload)
		})
	}

	var del DeleteEvent
	if err := json.Unmarshal(message, &del); err == nil && del.AvatarID != "" {
		return retry(ctx, 5, time.Second, func() error {
			return w.handleDelete(ctx, del)
		})
	}

	return fmt.Errorf("unknown event")
}

func (w *Worker) handleUpload(ctx context.Context, event UploadEvent) error {
	avatar, err := w.repo.GetAvatar(ctx, event.AvatarID)
	if err != nil {
		return err
	}

	if avatar.ProcessingStatus == "completed" {
		return nil
	}

	if avatar.ProcessingStatus != "processing" {
		if err := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, "processing"); err != nil {
			return err
		}
	}

	img, err := w.s3.Download(ctx, event.S3Key)
	if err != nil {
		return err
	}

	sizes := map[string]utils.ImageSize{
		"100x100": utils.Size100,
		"300x300": utils.Size300,
	}

	result := make(map[string]string)

	for sizeKey, size := range sizes {
		resized, _, err := utils.Process(img, size, utils.FormatJPEG)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("thumbnails/%s/%s.jpg", event.AvatarID, sizeKey)

		exists, err := w.s3.Exists(ctx, key)
		if err != nil {
			return err
		}

		if !exists {
			if err := w.s3.Upload(ctx, key, resized, "image/jpeg"); err != nil {
				return err
			}
		}

		result[sizeKey] = key
	}

	if err := w.repo.UpdateThumbnails(ctx, event.AvatarID, result); err != nil {
		return err
	}

	return w.repo.UpdateProcessingStatus(ctx, event.AvatarID, "completed")
}

func (w *Worker) handleDelete(ctx context.Context, event DeleteEvent) error {
	for _, key := range event.S3Keys {
		exists, err := w.s3.Exists(ctx, key)
		if err != nil {
			return err
		}

		if exists {
			if err := w.s3.Delete(ctx, key); err != nil {
				return err
			}
		}
	}

	return nil
}

func retry(ctx context.Context, attempts int, baseDelay time.Duration, fn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if i == attempts-1 {
			break
		}

		delay := baseDelay * time.Duration(1<<i)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}
