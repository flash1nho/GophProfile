package dto

import (
	"time"

	"github.com/flash1nho/GophProfile/internal/domain"
)

type AvatarUploadResponse struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	URL       string        `json:"url"`
	Status    domain.Status `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}
