package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/flash1nho/GophProfile/internal/repository"
	"github.com/flash1nho/GophProfile/pkg/storage"
	"github.com/flash1nho/GophProfile/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type repoIface interface {
	GetAvatar(ctx context.Context, id string) (*domain.Avatar, error)
	UpdateProcessingStatus(ctx context.Context, id string, status domain.ProcessingStatus) error
	UpdateThumbnails(ctx context.Context, id string, thumbs map[string]string) error
}

type s3Iface interface {
	Download(ctx context.Context, key string) ([]byte, error)
	Upload(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type Timer interface {
	After(d time.Duration) <-chan time.Time
}

type realTimer struct{}

type Worker struct {
	repo repoIface
	s3   s3Iface

	timer         Timer
	retryAttempts int
	retryDelay    time.Duration
}

func NewWorker(repo *repository.AvatarRepository, s3 *storage.S3) *Worker {
	return &Worker{
		repo:          repo,
		s3:            s3,
		timer:         realTimer{},
		retryAttempts: 5,
		retryDelay:    time.Second,
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

func (w *Worker) HandleUploadEvent(ctx context.Context, message []byte) error {
	ctx, span := otel.Tracer("worker").Start(ctx, "worker.handle_upload_event")
	defer span.End()

	span.SetAttributes(
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination", "avatars.queue"),
	)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var upload UploadEvent
	if err := json.Unmarshal(message, &upload); err == nil && upload.AvatarID != "" && upload.S3Key != "" {
		span.SetAttributes(
			attribute.String("avatar.id", upload.AvatarID),
			attribute.String("s3.key", upload.S3Key),
		)

		return retry(ctx, 5, time.Second, realTimer{}, func() error {
			return w.handleUpload(ctx, upload)
		})
	}

	var del DeleteEvent
	if err := json.Unmarshal(message, &del); err == nil && del.AvatarID != "" {
		return retry(ctx, 5, time.Second, realTimer{}, func() error {
			return w.handleDelete(ctx, del)
		})
	}

	err := fmt.Errorf("unknown event")

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

func (w *Worker) handleUpload(ctx context.Context, event UploadEvent) error {
	ctx, span := otel.Tracer("worker").Start(ctx, "worker.handle_upload")
	defer span.End()

	avatar, err := w.repo.GetAvatar(ctx, event.AvatarID)
	if err != nil {
		return fmt.Errorf("get avatar: %w", err)
	}

	span.SetAttributes(
		attribute.String("avatar.id", event.AvatarID),
		attribute.String("s3.key", event.S3Key),
	)

	if avatar.ProcessingStatus == domain.ProcessingStatusReady {
		return nil
	}

	if avatar.ProcessingStatus != domain.ProcessingStatusProcessing {
		if err := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, domain.ProcessingStatusProcessing); err != nil {
			return fmt.Errorf("set processing status: %w", err)
		}
	}

	img, err := w.s3.Download(ctx, event.S3Key)
	if err != nil {
		if errStatus := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, domain.ProcessingStatusFailed); errStatus != nil {
			return fmt.Errorf("download error: %v; additionally failed to update status: %w", err, errStatus)
		}
		return fmt.Errorf("download image: %w", err)
	}

	sizes := map[string]utils.ImageSize{
		"100x100": utils.Size100,
		"300x300": utils.Size300,
	}

	result := make(map[string]string)

	for sizeKey, size := range sizes {
		resized, _, err := utils.Process(img, size, utils.FormatJPEG)
		if err != nil {
			if errStatus := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, domain.ProcessingStatusFailed); errStatus != nil {
				return fmt.Errorf("process error: %v; status update failed: %w", err, errStatus)
			}
			return fmt.Errorf("process image: %w", err)
		}

		key := fmt.Sprintf("thumbnails/%s/%s.jpg", event.AvatarID, sizeKey)

		exists, err := w.s3.Exists(ctx, key)
		if err != nil {
			if errStatus := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, domain.ProcessingStatusFailed); errStatus != nil {
				return fmt.Errorf("exists check error: %v; status update failed: %w", err, errStatus)
			}
			return fmt.Errorf("check exists: %w", err)
		}

		if !exists {
			if err := w.s3.Upload(ctx, key, resized, "image/jpeg"); err != nil {
				if errStatus := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, domain.ProcessingStatusFailed); errStatus != nil {
					return fmt.Errorf("upload error: %v; status update failed: %w", err, errStatus)
				}
				return fmt.Errorf("upload thumbnail: %w", err)
			}
		}

		result[sizeKey] = key
	}

	if err := w.repo.UpdateThumbnails(ctx, event.AvatarID, result); err != nil {
		if errStatus := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, domain.ProcessingStatusFailed); errStatus != nil {
			return fmt.Errorf("update thumbnails error: %v; status update failed: %w", err, errStatus)
		}
		return fmt.Errorf("update thumbnails: %w", err)
	}

	if err := w.repo.UpdateProcessingStatus(ctx, event.AvatarID, domain.ProcessingStatusReady); err != nil {
		return fmt.Errorf("set ready status: %w", err)
	}

	return nil
}

func (w *Worker) handleDelete(ctx context.Context, event DeleteEvent) error {
	for _, key := range event.S3Keys {

		exists, err := w.s3.Exists(ctx, key)
		if err != nil {
			return fmt.Errorf("check exists (%s): %w", key, err)
		}

		if exists {
			if err := w.s3.Delete(ctx, key); err != nil {
				return fmt.Errorf("delete (%s): %w", key, err)
			}
		}
	}

	return nil
}

func (r realTimer) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func retry(ctx context.Context, attempts int, baseDelay time.Duration, t Timer, fn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {

		if ctx.Err() != nil {
			return fmt.Errorf("retry canceled: %w", ctx.Err())
		}

		err = fn()
		if err == nil {
			return nil
		}

		if i == attempts-1 {
			break
		}

		delay := baseDelay * time.Duration(1<<i)

		select {
		case <-t.After(delay):
		case <-ctx.Done():
			return fmt.Errorf("retry canceled: %w", ctx.Err())
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", attempts, err)
}
