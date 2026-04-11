package handlers

import (
	"io"
	"net/http"

	"github.com/flash1nho/GophProfile/internal/services"
	httpresp "github.com/flash1nho/GophProfile/pkg/http"
)

type AvatarHandler struct {
	svc *services.AvatarService
}

func NewAvatarHandler(s *services.AvatarService) *AvatarHandler {
	return &AvatarHandler{svc: s}
}

func (h *AvatarHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		httpresp.Error(w, 400, "missing user id")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpresp.Error(w, 400, "invalid file")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		httpresp.Error(w, 500, "failed to read file")
		return
	}

	if len(data) == 0 {
		httpresp.Error(w, 400, "empty file")
		return
	}

	avatar, err := h.svc.Upload(
		r.Context(),
		userID,
		header.Filename,
		header.Header.Get("Content-Type"),
		data,
	)
	if err != nil {
		httpresp.Error(w, 500, err.Error())
		return
	}

	httpresp.JSON(w, 201, avatar)
}
