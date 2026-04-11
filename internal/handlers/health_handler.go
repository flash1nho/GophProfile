package handlers

import (
	"context"
	"net/http"
	"time"

	httpresp "github.com/flash1nho/GophProfile/pkg/http"
)

func (h *AvatarHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status := map[string]string{
		"status": "ok",
	}

	if err := h.svc.PingDB(ctx); err != nil {
		status["db"] = "error"
	}

	if err := h.svc.PingS3(ctx); err != nil {
		status["s3"] = "error"
	}

	if err := h.svc.PingRabbit(ctx); err != nil {
		status["rabbit"] = "error"
	}

	httpresp.JSON(w, http.StatusOK, status)
}
