package observability

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ctxKey int

const (
	loggerKey ctxKey = iota
	requestIDKey
)

func InjectLogger(ctx context.Context, base *zap.Logger) context.Context {
	if base == nil {
		return ctx
	}

	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	logger := base

	if spanCtx.IsValid() {
		logger = base.With(
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
	}

	if reqID := RequestIDFromContext(ctx); reqID != "" {
		logger = logger.With(zap.String("request_id", reqID))
	}

	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.L()
}

func InjectRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
