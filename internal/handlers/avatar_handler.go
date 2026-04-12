package handlers

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flash1nho/GophProfile/internal/dto"
	"github.com/flash1nho/GophProfile/internal/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	httpresp "github.com/flash1nho/GophProfile/pkg/http"
	"github.com/flash1nho/GophProfile/pkg/utils"
)

const maxFileSize = 10 << 20 // 10MB

type AvatarHandler struct {
	svc   *services.AvatarService
	log   *zap.Logger
	cache Cache
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

func NewAvatarHandler(svc *services.AvatarService, log *zap.Logger, cache Cache) *AvatarHandler {
	return &AvatarHandler{
		svc:   svc,
		log:   log,
		cache: cache,
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
			httpresp.JSON(w, http.StatusRequestEntityTooLarge, map[string]string{
				"error":    "File too large",
				"max_size": fmt.Sprintf("%d", maxFileSize),
			})
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
		httpresp.JSON(w, http.StatusBadRequest, map[string]string{
			"error":   "Invalid file format",
			"details": "Supported formats: jpeg, png, webp",
		})
		return
	}

	avatar, err := h.svc.Upload(ctx, userID, header.Filename, mime, data)
	if err != nil {
		httpresp.Error(w, 500, err.Error())
		return
	}

	baseURL := getBaseURL(r)

	response := dto.AvatarUploadResponse{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		URL:       fmt.Sprintf("%s/api/v1/avatars/%s?format=webp", baseURL, avatar.ID),
		Status:    avatar.UploadStatus,
		CreatedAt: *avatar.CreatedAt,
	}

	httpresp.JSON(w, http.StatusCreated, response)
}

func (h *AvatarHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	sizeParam := r.URL.Query().Get("size")
	if sizeParam == "" {
		sizeParam = "original"
	}

	formatParam := r.URL.Query().Get("format")
	if formatParam == "" {
		formatParam = "jpeg"
	}

	var size utils.ImageSize
	switch sizeParam {
	case "100x100":
		size = utils.Size100
	case "300x300":
		size = utils.Size300
	case "original":
		size = utils.SizeOriginal
	default:
		httpresp.Error(w, 400, "invalid size")
		return
	}

	var format utils.ImageFormat
	switch formatParam {
	case "jpeg":
		format = utils.FormatJPEG
	case "png":
		format = utils.FormatPNG
	case "webp":
		format = utils.FormatWEBP
	default:
		httpresp.Error(w, 400, "invalid format")
		return
	}

	cacheKey := fmt.Sprintf("avatar:%s:%s:%s", id, sizeParam, formatParam)

	if h.cache != nil {
		if cached, err := h.cache.Get(r.Context(), cacheKey); err == nil && cached != nil {
			mime := "image/jpeg"
			switch formatParam {
			case "png":
				mime = "image/png"
			case "webp":
				mime = "image/webp"
			}

			hash := sha1.Sum(cached)
			etag := fmt.Sprintf(`"%x"`, hash)

			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Content-Type", mime)
			w.Header().Set("Cache-Control", "max-age=86400")
			w.Header().Set("ETag", etag)

			_, err := w.Write(cached)

			if err != nil {
				h.log.Error("failed to write http cache", zap.Error(err))
			}

			return
		}
	}

	data, _, err := h.svc.Get(r.Context(), id, "original")
	if err != nil {
		httpresp.Error(w, 404, "Avatar not found")
		return
	}

	data, mime, err := utils.Process(data, size, format)
	if err != nil {
		httpresp.Error(w, 500, "failed to process image")
		return
	}

	if h.cache != nil {
		err = h.cache.Set(r.Context(), cacheKey, data, 24*time.Hour)

		if err != nil {
			h.log.Error("failed to SET cache", zap.String("key", cacheKey))
		} else {
			h.log.Info("cache SET", zap.String("key", cacheKey))
		}
	}

	hash := sha1.Sum(data)
	etag := fmt.Sprintf(`"%x"`, hash)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Header().Set("ETag", etag)

	_, err = w.Write(data)
	if err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
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
	avatarID := chi.URLParam(r, "id")
	if avatarID == "" {
		httpresp.Error(w, http.StatusBadRequest, "avatar_id is required")
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		httpresp.Error(w, http.StatusBadRequest, "X-User-ID header is required")
		return
	}

	err := h.svc.Delete(r.Context(), avatarID, userID)
	if err != nil {
		if err.Error() == "forbidden" {
			httpresp.JSON(w, http.StatusForbidden, map[string]string{
				"error":   "Forbidden",
				"details": "You can only delete your own avatars",
			})
			return
		}

		httpresp.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AvatarHandler) DeleteByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		httpresp.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}

	headerID := r.Header.Get("X-User-ID")
	if headerID == "" {
		httpresp.Error(w, http.StatusBadRequest, "X-User-ID header is required")
		return
	}

	if userID != headerID {
		httpresp.JSON(w, http.StatusForbidden, map[string]string{
			"error":   "Forbidden",
			"details": "You can only delete your own avatars",
		})
		return
	}

	if err := h.svc.DeleteByUser(r.Context(), userID); err != nil {
		httpresp.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AvatarHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	avatar, err := h.svc.GetMetadata(r.Context(), id)
	if err != nil {
		httpresp.Error(w, 404, "Avatar not found")
		return
	}

	baseURL := getBaseURL(r)

	thumbnails := []dto.ThumbnailDTO{
		{
			Size: "100x100",
			URL:  fmt.Sprintf("%s/api/v1/avatars/%s?size=100x100&format=webp", baseURL, avatar.ID),
		},
		{
			Size: "300x300",
			URL:  fmt.Sprintf("%s/api/v1/avatars/%s?size=300x300&format=webp", baseURL, avatar.ID),
		},
	}

	data, _, err := h.svc.Get(r.Context(), id, "original")
	if err != nil {
		httpresp.Error(w, 500, "failed to load image")
		return
	}

	width, height, err := utils.GetDimensions(data)
	if err != nil {
		width = 0
		height = 0
	}

	resp := dto.AvatarMetadataResponse{
		ID:       avatar.ID,
		UserID:   avatar.UserID,
		FileName: avatar.FileName,
		MimeType: avatar.MimeType,
		Size:     avatar.SizeBytes,
		Dimensions: dto.DimensionsDTO{
			Width:  width,
			Height: height,
		},
		Thumbnails: thumbnails,
		CreatedAt:  *avatar.CreatedAt,
		UpdatedAt:  *avatar.UpdatedAt,
	}

	httpresp.JSON(w, http.StatusOK, resp)
}

func (h *AvatarHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	avatars, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		httpresp.Error(w, 500, err.Error())
		return
	}

	baseURL := getBaseURL(r)

	result := make([]dto.AvatarUploadResponse, 0, len(avatars))

	for _, a := range avatars {
		result = append(result, dto.AvatarUploadResponse{
			ID:        a.ID,
			UserID:    a.UserID,
			URL:       fmt.Sprintf("%s/api/v1/avatars/%s?format=webp", baseURL, a.ID),
			Status:    a.UploadStatus,
			CreatedAt: *a.CreatedAt,
		})
	}

	httpresp.JSON(w, http.StatusOK, result)
}

func getBaseURL(r *http.Request) string {
	scheme := "http"

	if r.TLS != nil {
		scheme = "https"
	}

	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
