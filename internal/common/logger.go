package common

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

func NewLogger(service string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", service).Logger()
	return logger
}

func WithContext(ctx context.Context, logger zerolog.Logger) zerolog.Logger {
	if ctx == nil {
		return logger
	}
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		logger = logger.With().Str("trace_id", sc.TraceID().String()).Logger()
	}
	if sc.HasSpanID() {
		logger = logger.With().Str("span_id", sc.SpanID().String()).Logger()
	}
	return logger
}
