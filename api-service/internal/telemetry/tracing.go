// File: api-service/internal/telemetry/tracing.go
package telemetry

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0" // Use the latest appropriate version
)

// InitTracerProvider initializes and returns a new OpenTelemetry TracerProvider.
func InitTracerProvider(endpoint string) (*trace.TracerProvider, error) {
	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint), // Now uses the injected variable
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// Create a new resource to identify this application
	// "go-api" will show up in Grafana
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("go-api"),
			semconv.ServiceVersion("1.0.1"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create the TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
	)

	// Set the global TracerProvider
	otel.SetTracerProvider(tp)

	log.Println("OpenTelemetry TracerProvider initialized, sending to http://tempo:4318")
	return tp, nil
}
