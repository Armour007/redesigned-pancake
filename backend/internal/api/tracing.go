package api

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupOTelFromEnv initializes OpenTelemetry tracing if enabled by env.
// Controlled by AURA_OTEL_ENABLE=true and OTEL_EXPORTER_OTLP_ENDPOINT.
// Returns a shutdown func that should be deferred by the caller.
func SetupOTelFromEnv() (func(context.Context) error, bool) {
	if os.Getenv("AURA_OTEL_ENABLE") == "" && os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return func(ctx context.Context) error { return nil }, false
	}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// sensible default
		endpoint = "http://localhost:4318"
	}
	// Use HTTP exporter for minimal deps
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	exp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		log.Printf("otel exporter init failed: %v", err)
		return func(ctx context.Context) error { return nil }, false
	}
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("aura-backend"),
		),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	shutdown := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}
	return shutdown, true
}
