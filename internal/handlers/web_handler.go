package handlers

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *AvatarHandler) WebUploadForm(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/upload.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (h *AvatarHandler) WebUploadSubmit(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", 400)
		return
	}

	r.Header.Set("X-User-ID", userID)

	h.Upload(w, r)
}

func (h *AvatarHandler) WebGallery(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	avatars, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	tmpl, err := template.ParseFiles("web/gallery.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := tmpl.Execute(w, avatars); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
