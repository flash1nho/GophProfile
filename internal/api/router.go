package api

import (
	"net/http"

	"github.com/flash1nho/GophProfile/internal/handlers"
	"github.com/go-chi/chi/v5"
)

func NewRouter(h *handlers.AvatarHandler) http.Handler {
	r := chi.NewRouter()

	r.Post("/api/v1/avatars", h.Upload)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
