package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"go.uber.org/zap"

	"github.com/flash1nho/GophProfile/internal/dto"
	"github.com/go-chi/chi/v5"
)

func (h *AvatarHandler) WebUploadForm(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/static/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (h *AvatarHandler) WebUploadSubmit(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	if userID == "" {
		userID = r.FormValue("user_id")
	}

	if userID == "" {
		http.Error(w, "missing user_id", 400)
		return
	}

	h.Upload(w, r)
}

func (h *AvatarHandler) WebGallery(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	avatars, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	baseURL := getBaseURL(r)

	items := make([]dto.GalleryItem, 0, len(avatars))

	for _, a := range avatars {
		items = append(items, dto.GalleryItem{
			ID:       a.ID,
			FileName: a.FileName,
			UserID:   userID,
			URL: fmt.Sprintf(
				"%s/api/v1/avatars/%s?size=300x300&format=webp",
				baseURL,
				a.ID,
			),
		})
	}

	tmpl, err := template.ParseFiles("web/static/gallery.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := tmpl.Execute(w, items); err != nil {
		h.log.Error("template error", zap.Error(err))
		http.Error(w, err.Error(), 500)
	}
}
