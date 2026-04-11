package domain

import "time"

type Avatar struct {
	ID               string
	UserID           string
	FileName         string
	MimeType         string
	SizeBytes        int64
	S3Key            string
	ThumbnailKeys    map[string]string
	UploadStatus     string
	ProcessingStatus string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
