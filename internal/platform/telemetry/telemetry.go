// Package telemetry sets up OpenTelemetry tracing for a service.
//
// The OTLP exporter reads OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_INSECURE and the
// OTEL_EXPORTER_OTLP_TRACES_* variants from the environment. Resource attributes come from
// OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config controls whether tracing is exported and which service identity is reported.
type Config struct {
	Enabled        bool
	ServiceName    string // Overrides OTEL_SERVICE_NAME when non-empty.
	ServiceVersion string
}

// ShutdownFunc flushes pending spans and releases exporter resources.
type ShutdownFunc func(ctx context.Context) error

// Setup installs the global W3C TraceContext and Baggage propagators and, when cfg.Enabled is true,
// a tracer provider with a batch span processor and an OTLP gRPC exporter.
//
// The propagators are installed even when tracing is disabled, so trace headers still pass
// through the service. When disabled, Setup returns a no-op ShutdownFunc and never contacts a collector.
func Setup(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create otlp trace exporter: %w", err)
	}

	// resource.Merge gives later options precedence, so explicit config overrides the environment.
	var identity []attribute.KeyValue
	if cfg.ServiceName != "" {
		identity = append(identity, attribute.String("service.name", cfg.ServiceName))
	}
	if cfg.ServiceVersion != "" {
		identity = append(identity, attribute.String("service.version", cfg.ServiceVersion))
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(identity...),
	)
	if err != nil {
		_ = exporter.Shutdown(ctx)
		return nil, fmt.Errorf("telemetry: create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
