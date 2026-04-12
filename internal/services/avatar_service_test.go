package services

import (
	"context"
	"errors"
	"testing"

	"github.com/flash1nho/GophProfile/internal/domain"
	"go.uber.org/zap"
)

type mockRepo struct {
	createErr error

	getAvatar *domain.Avatar
	getErr    error

	updateUploadErr     error
	updateProcessingErr error

	deleteErr error

	latest *domain.Avatar
	list   []domain.Avatar
}

func (m *mockRepo) Create(ctx context.Context, a *domain.Avatar) error {
	return m.createErr
}

func (m *mockRepo) GetAvatar(ctx context.Context, id string) (*domain.Avatar, error) {
	return m.getAvatar, m.getErr
}

func (m *mockRepo) SoftDelete(ctx context.Context, id string) error {
	return m.deleteErr
}

func (m *mockRepo) GetLatestByUser(ctx context.Context, userID string) (*domain.Avatar, error) {
	return m.latest, nil
}

func (m *mockRepo) ListByUser(ctx context.Context, userID string) ([]domain.Avatar, error) {
	return m.list, nil
}

func (m *mockRepo) UpdateUploadStatus(ctx context.Context, id string, status domain.UploadStatus) error {
	return m.updateUploadErr
}

func (m *mockRepo) UpdateProcessingStatus(ctx context.Context, id string, status domain.ProcessingStatus) error {
	return m.updateProcessingErr
}

func (m *mockRepo) UpdateThumbnails(ctx context.Context, id string, thumbs map[string]string) error {
	return nil
}

func (m *mockRepo) Ping(ctx context.Context) error { return nil }

type mockStorage struct {
	uploadErr   error
	downloadErr error
	deleteErr   error

	data []byte
}

func (m *mockStorage) Upload(ctx context.Context, key string, data []byte, mime string) error {
	return m.uploadErr
}

func (m *mockStorage) Download(ctx context.Context, key string) ([]byte, error) {
	return m.data, m.downloadErr
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	return m.deleteErr
}

func (m *mockStorage) Health(ctx context.Context) error { return nil }

type mockPublisher struct {
	err error
}

func (m *mockPublisher) Publish(v any) error { return m.err }
func (m *mockPublisher) Ping() error         { return nil }

func TestUpload_AllBranches(t *testing.T) {
	tests := []struct {
		name string

		repoErr    error
		s3Err      error
		statusErr  error
		publishErr error

		expectError bool
		expectNil   bool
	}{
		{
			name: "success",
		},
		{
			name:        "repo create error",
			repoErr:     errors.New("db fail"),
			expectError: true,
			expectNil:   true,
		},
		{
			name:        "s3 upload error",
			s3Err:       errors.New("s3 fail"),
			expectError: true,
			expectNil:   true,
		},
		{
			name:        "status update error",
			statusErr:   errors.New("status fail"),
			expectError: true,
			expectNil:   true,
		},
		{
			name:        "publish error (partial success)",
			publishErr:  errors.New("mq fail"),
			expectError: true,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				createErr:           tt.repoErr,
				updateUploadErr:     tt.statusErr,
				updateProcessingErr: tt.statusErr,
			}

			s3 := &mockStorage{
				uploadErr: tt.s3Err,
			}

			pub := &mockPublisher{
				err: tt.publishErr,
			}

			svc := NewAvatarService(repo, s3, pub, zap.NewNop())

			avatar, err := svc.Upload(
				context.Background(),
				"user1",
				"file.jpg",
				"image/jpeg",
				[]byte("data"),
			)

			if tt.expectError && err == nil {
				t.Fatalf("expected error")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectNil && avatar != nil {
				t.Fatalf("expected nil avatar")
			}

			if !tt.expectNil && avatar == nil {
				t.Fatalf("expected avatar")
			}
		})
	}
}

func TestGet_AllCases(t *testing.T) {
	avatar := &domain.Avatar{
		S3Key:    "original",
		MimeType: "image/jpeg",
		ThumbnailKeys: map[string]string{
			"100x100": "thumb100",
		},
	}

	tests := []struct {
		name string
		size string

		repoErr     error
		storageErr  error
		expectError bool
	}{
		{
			name: "original",
		},
		{
			name: "thumbnail",
			size: "100x100",
		},
		{
			name:        "repo error",
			repoErr:     errors.New("fail"),
			expectError: true,
		},
		{
			name:        "storage error",
			storageErr:  errors.New("fail"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				getAvatar: avatar,
				getErr:    tt.repoErr,
			}

			s3 := &mockStorage{
				data:        []byte("img"),
				downloadErr: tt.storageErr,
			}

			svc := NewAvatarService(repo, s3, nil, zap.NewNop())

			_, _, err := svc.Get(context.Background(), "id", tt.size)

			if tt.expectError && err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestDelete(t *testing.T) {
	repo := &mockRepo{
		getAvatar: &domain.Avatar{
			ID:     "1",
			UserID: "user1",
		},
	}

	svc := NewAvatarService(repo, nil, nil, zap.NewNop())

	err := svc.Delete(context.Background(), "1", "user1")
	if err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestDelete_Forbidden(t *testing.T) {
	repo := &mockRepo{
		getAvatar: &domain.Avatar{
			UserID: "other",
		},
	}

	svc := NewAvatarService(repo, nil, nil, zap.NewNop())

	err := svc.Delete(context.Background(), "1", "user1")

	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteByUser(t *testing.T) {
	repo := &mockRepo{
		latest: &domain.Avatar{
			ID:     "1",
			UserID: "user1",
		},
		getAvatar: &domain.Avatar{
			ID:     "1",
			UserID: "user1",
		},
	}

	svc := NewAvatarService(repo, nil, nil, zap.NewNop())

	err := svc.DeleteByUser(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSimpleMethods(t *testing.T) {
	repo := &mockRepo{
		getAvatar: &domain.Avatar{},
		latest:    &domain.Avatar{},
		list:      []domain.Avatar{{}},
	}

	svc := NewAvatarService(repo, nil, nil, zap.NewNop())

	_, _ = svc.GetByUser(context.Background(), "user1")
	_, _ = svc.ListByUser(context.Background(), "user1")
	_, _ = svc.GetMetadata(context.Background(), "id")
}

func TestPing(t *testing.T) {
	repo := &mockRepo{}
	s3 := &mockStorage{}
	pub := &mockPublisher{}

	svc := NewAvatarService(repo, s3, pub, zap.NewNop())

	if err := svc.PingDB(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := svc.PingS3(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := svc.PingRabbit(context.Background()); err != nil {
		t.Fatal(err)
	}
}
