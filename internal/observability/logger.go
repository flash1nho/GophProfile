package observability

import (
    "context"

    "go.opentelemetry.io/otel/trace"
    "go.uber.org/zap"
)

func WithTrace(ctx context.Context, log *zap.Logger) *zap.Logger {
    span := trace.SpanFromContext(ctx)

    if !span.SpanContext().IsValid() {
        return log
    }

    return log.With(
        zap.String("trace_id", span.SpanContext().TraceID().String()),
    )
}
