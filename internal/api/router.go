package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/flash1nho/GophProfile/internal/handlers"
	"github.com/flash1nho/GophProfile/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRouter(h *handlers.AvatarHandler, log *zap.Logger) http.Handler {
	r := chi.NewRouter()

	rl := middleware.NewRateLimiter(5, 10)

	r.Use(middleware.CORS)
	r.Use(rl.Middleware)
	r.Use(middleware.Logger(log))

	r.Use(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http-request")
	})

	r.Post("/api/v1/avatars", h.Upload)
	r.With(middleware.RequireUser).Delete("/api/v1/avatars/{id}", h.Delete)

	r.Get("/api/v1/avatars/{id}", h.Get)
	r.Get("/api/v1/avatars/{id}/metadata", h.Metadata)

	r.Get("/api/v1/users/{user_id}/avatar", h.GetByUser)
	r.With(middleware.RequireUser).Delete("/api/v1/users/{user_id}/avatar", h.DeleteByUser)
	r.Get("/api/v1/users/{user_id}/avatars", h.ListByUser)

	r.Get("/health", h.Health)

	r.Get("/web/upload", h.WebUploadForm)
	r.Post("/web/upload", h.WebUploadSubmit)
	r.Get("/web/gallery/{user_id}", h.WebGallery)

	r.Handle("/metrics", promhttp.Handler())

	return r
}
