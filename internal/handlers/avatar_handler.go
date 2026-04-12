package handlers

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"

	"github.com/flash1nho/GophProfile/internal/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	httpresp "github.com/flash1nho/GophProfile/pkg/http"
)

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

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httpresp.Error(w, 413, "file too large")
		return
	}

	file, header, err := r.FormFile("image")
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

	avatar, err := h.svc.Upload(ctx, userID, header.Filename, mime, data)
	if err != nil {
		httpresp.Error(w, 500, err.Error())
		return
	}

	httpresp.JSON(w, 201, avatar)
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
