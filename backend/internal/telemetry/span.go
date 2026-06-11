package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/LiamYaoLian/stripe-payment-integration-prototype/backend"

var tracer = otel.Tracer(tracerName)

// Start begins a child span on the shared service tracer.
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, opts...)
}

// StringAttr is a small helper for span attributes.
func StringAttr(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// LogFields returns slog key/value pairs for the active trace, if any.
func LogFields(ctx context.Context) []any {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []any{
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	}
}
