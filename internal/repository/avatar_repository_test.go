package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flash1nho/GophProfile/internal/domain"
	pgxmock "github.com/pashagolub/pgxmock/v3"
	"go.uber.org/zap"
)

func newRepo(t *testing.T) (*AvatarRepository, pgxmock.PgxPoolIface) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	repo := NewAvatarRepository(mock, zap.NewNop())
	return repo, mock
}

func TestCreate_OK(t *testing.T) {
	repo, mock := newRepo(t)

	a := &domain.Avatar{
		ID:       "1",
		UserID:   "user1",
		FileName: "file.jpg",
		MimeType: "image/jpeg",
		S3Key:    "key",
	}

	mock.ExpectQuery("INSERT INTO avatars").
		WithArgs(a.ID, a.UserID, a.FileName, a.MimeType, a.SizeBytes, a.S3Key).
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

	err := repo.Create(context.Background(), a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreate_Error(t *testing.T) {
	repo, mock := newRepo(t)

	a := &domain.Avatar{}

	mock.ExpectQuery("INSERT INTO avatars").
		WithArgs(a.ID, a.UserID, a.FileName, a.MimeType, a.SizeBytes, a.S3Key).
		WillReturnError(errors.New("db error"))

	err := repo.Create(context.Background(), a)
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetAvatar_OK(t *testing.T) {
	repo, mock := newRepo(t)

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "file_name", "mime_type",
		"size_bytes", "s3_key", "upload_status",
		"processing_status", "created_at", "updated_at",
	}).AddRow(
		"1", "user1", "file.jpg", "image/jpeg",
		int64(100), "key", "uploaded", "done", time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT .* FROM avatars").
		WithArgs("1").
		WillReturnRows(rows)

	a, err := repo.GetAvatar(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.ID != "1" {
		t.Fatalf("wrong id")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetAvatar_Error(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectQuery("SELECT .* FROM avatars").
		WithArgs("1").
		WillReturnError(errors.New("not found"))

	_, err := repo.GetAvatar(context.Background(), "1")
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSoftDelete(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectExec("UPDATE avatars").
		WithArgs("1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.SoftDelete(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetLatestByUser_OK(t *testing.T) {
	repo, mock := newRepo(t)

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "file_name", "mime_type", "size_bytes", "s3_key",
	}).AddRow("1", "user1", "file.jpg", "image/jpeg", int64(100), "key")

	mock.ExpectQuery("SELECT .* FROM avatars").
		WithArgs("user1").
		WillReturnRows(rows)

	a, err := repo.GetLatestByUser(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	if a.UserID != "user1" {
		t.Fatalf("wrong user")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListByUser_OK(t *testing.T) {
	repo, mock := newRepo(t)
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "file_name", "mime_type",
		"size_bytes", "s3_key", "upload_status",
		"processing_status", "created_at", "updated_at",
	}).AddRow(
		"1", "user1", "file.jpg", "image/jpeg",
		int64(100), "key", domain.UploadStatus("uploaded"), domain.ProcessingStatus("ready"), &now, &now,
	)

	mock.ExpectQuery("SELECT .* FROM avatars").
		WithArgs("user1").
		WillReturnRows(rows)

	list, err := repo.ListByUser(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListByUser_QueryError(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectQuery("SELECT .* FROM avatars").
		WithArgs("user1").
		WillReturnError(errors.New("db error"))

	_, err := repo.ListByUser(context.Background(), "user1")
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateUploadStatus(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectExec("UPDATE avatars SET upload_status").
		WithArgs("uploaded", "1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateUploadStatus(context.Background(), "1", "uploaded")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateProcessingStatus(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectExec("UPDATE avatars SET processing_status").
		WithArgs("done", "1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateProcessingStatus(context.Background(), "1", "done")
	if err != nil {
		t.Fatalf("unexpected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateThumbnails(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectExec("UPDATE avatars").
		WithArgs(map[string]string{"100x100": "key"}, "1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateThumbnails(context.Background(), "1", map[string]string{
		"100x100": "key",
	})
	if err != nil {
		t.Fatalf("unexpected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPing(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectPing()

	err := repo.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
