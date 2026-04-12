package handlers

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flash1nho/GophProfile/internal/dto"
	"github.com/flash1nho/GophProfile/internal/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	httpresp "github.com/flash1nho/GophProfile/pkg/http"
)

const maxFileSize = 10 << 20 // 10MB

type AvatarHandler struct {
	svc *services.AvatarService
	log *zap.Logger
}

func NewAvatarHandler(svc *services.AvatarService, log *zap.Logger) *AvatarHandler {
	return &AvatarHandler{
		svc: svc,
		log: log,
	}
}

func (h *AvatarHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		httpresp.Error(w, 400, "missing X-User-ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "File too large",
				"max_size": maxFileSize,
			}); err != nil {
				h.log.Error("encode error", zap.Error(err))
			}
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpresp.Error(w, 400, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		httpresp.Error(w, 500, "read error")
		return
	}

	mime := http.DetectContentType(data)

	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}

	if !allowedTypes[mime] {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid file format",
			"details": "Supported formats: jpeg, png, webp",
		}); err != nil {
			h.log.Error("encode error", zap.Error(err))
		}
		return
	}

	avatar, err := h.svc.Upload(ctx, userID, header.Filename, mime, data)
	if err != nil {
		httpresp.Error(w, 500, err.Error())
		return
	}

	response := dto.AvatarUploadResponse{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		URL:       fmt.Sprintf("/api/v1/avatars/%s", avatar.ID),
		Status:    avatar.UploadStatus,
		CreatedAt: avatar.CreatedAt,
	}

	httpresp.JSON(w, http.StatusCreated, response)
}

func (h *AvatarHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	data, mime, err := h.svc.Get(r.Context(), id, "original")
	if err != nil {
		httpresp.Error(w, 404, "not found")
		return
	}

	hash := sha1.Sum(data)
	etag := fmt.Sprintf(`"%x"`, hash)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(304)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("ETag", etag)

	_, err = w.Write(data)
	if err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

func (h *AvatarHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	avatar, err := h.svc.GetByUser(r.Context(), userID)
	if err != nil {
		httpresp.Error(w, 404, "not found")
		return
	}

	http.Redirect(w, r, "/api/v1/avatars/"+avatar.ID, 302)
}

func (h *AvatarHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := r.Header.Get("X-User-ID")

	if err := h.svc.Delete(r.Context(), id, userID); err != nil {
		httpresp.Error(w, 403, err.Error())
		return
	}

	w.WriteHeader(204)
}

func (h *AvatarHandler) DeleteByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	headerID := r.Header.Get("X-User-ID")

	if userID != headerID {
		httpresp.Error(w, 403, "forbidden")
		return
	}

	if err := h.svc.DeleteByUser(r.Context(), userID); err != nil {
		httpresp.Error(w, 500, err.Error())
		return
	}

	w.WriteHeader(204)
}

func (h *AvatarHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	meta, err := h.svc.GetMetadata(r.Context(), id)
	if err != nil {
		httpresp.Error(w, 404, "not found")
		return
	}

	httpresp.JSON(w, 200, meta)
}

func (h *AvatarHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	list, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		httpresp.Error(w, 500, err.Error())
		return
	}

	httpresp.JSON(w, 200, list)
}
