package handlers

import (
	"context"
	"net/http"
	"time"

	httpresp "github.com/flash1nho/GophProfile/pkg/http"
	"go.uber.org/zap"
)

func (h *AvatarHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status := map[string]string{
		"db":     "ok",
		"s3":     "ok",
		"rabbit": "ok",
	}

	overallStatus := "ok"
	httpStatus := http.StatusOK

	if err := h.svc.PingDB(ctx); err != nil {
		h.log.Error("db healthcheck failed", zap.Error(err))
		status["db"] = "error"
		overallStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	if err := h.svc.PingS3(ctx); err != nil {
		h.log.Error("s3 healthcheck failed", zap.Error(err))
		status["s3"] = "error"
		overallStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	if err := h.svc.PingRabbit(ctx); err != nil {
		h.log.Error("rabbit healthcheck failed", zap.Error(err))
		status["rabbit"] = "error"
		overallStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	status["status"] = overallStatus

	httpresp.JSON(w, httpStatus, status)
}
