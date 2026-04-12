package dto

import (
	"time"

	"github.com/flash1nho/GophProfile/internal/domain"
)

type AvatarUploadResponse struct {
	ID        string              `json:"id"`
	UserID    string              `json:"user_id"`
	URL       string              `json:"url"`
	Status    domain.UploadStatus `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
}

type DimensionsDTO struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ThumbnailDTO struct {
	Size string `json:"size"`
	URL  string `json:"url"`
}

type AvatarMetadataResponse struct {
	ID         string         `json:"id"`
	UserID     string         `json:"user_id"`
	FileName   string         `json:"file_name"`
	MimeType   string         `json:"mime_type"`
	Size       int64          `json:"size"`
	Dimensions DimensionsDTO  `json:"dimensions"`
	Thumbnails []ThumbnailDTO `json:"thumbnails"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type GalleryItem struct {
	ID       string
	FileName string
	URL      string
}
