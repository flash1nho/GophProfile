package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/flash1nho/GophProfile/internal/domain"
	"github.com/flash1nho/GophProfile/internal/services"
)

type mockRepo struct {
	err error
}

func (m *mockRepo) Ping(ctx context.Context) error {
	return m.err
}

func (m *mockRepo) Create(ctx context.Context, a *domain.Avatar) error {
	return nil
}

func (m *mockRepo) GetAvatar(ctx context.Context, id string) (*domain.Avatar, error) {
	return nil, nil
}

func (m *mockRepo) SoftDelete(ctx context.Context, id string) error {
	return nil
}

func (m *mockRepo) GetLatestByUser(ctx context.Context, userID string) (*domain.Avatar, error) {
	return nil, nil
}

func (m *mockRepo) ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error) {
	return nil, nil
}

func (m *mockRepo) UpdateUploadStatus(ctx context.Context, id string, status domain.UploadStatus) error {
	return nil
}

func (m *mockRepo) UpdateProcessingStatus(ctx context.Context, id string, status domain.ProcessingStatus) error {
	return nil
}

func (m *mockRepo) UpdateThumbnails(ctx context.Context, id string, thumbs map[string]string) error {
	return nil
}

type mockS3 struct {
	err error
}

func (m *mockS3) Health(ctx context.Context) error { return m.err }
func (m *mockS3) Upload(context.Context, string, []byte, string) error {
	return nil
}
func (m *mockS3) Download(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (m *mockS3) Delete(context.Context, string) error { return nil }

type mockPub struct {
	err error
}

func (m *mockPub) Ping() error       { return m.err }
func (m *mockPub) Publish(any) error { return nil }

func TestHealth_AllOK(t *testing.T) {
	svc := services.NewAvatarService(
		&mockRepo{},
		&mockS3{},
		&mockPub{},
		zap.NewNop(),
	)

	h := &AvatarHandler{
		svc: svc,
		log: zap.NewNop(),
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHealth_DBError(t *testing.T) {
	svc := services.NewAvatarService(
		&mockRepo{err: errors.New("db down")},
		&mockS3{},
		&mockPub{},
		zap.NewNop(),
	)

	h := &AvatarHandler{svc: svc, log: zap.NewNop()}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503")
	}
}

func TestHealth_S3Error(t *testing.T) {
	svc := services.NewAvatarService(
		&mockRepo{},
		&mockS3{err: errors.New("s3 down")},
		&mockPub{},
		zap.NewNop(),
	)

	h := &AvatarHandler{svc: svc, log: zap.NewNop()}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503")
	}
}

func TestHealth_RabbitError(t *testing.T) {
	svc := services.NewAvatarService(
		&mockRepo{},
		&mockS3{},
		&mockPub{err: errors.New("rabbit down")},
		zap.NewNop(),
	)

	h := &AvatarHandler{svc: svc, log: zap.NewNop()}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503")
	}
}
