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
func InitTracerProvider() (*trace.TracerProvider, error) {
	ctx := context.Background()

	// Configure an OTLP/HTTP exporter to send traces to Tempo.
	// The endpoint matches the Tempo service in your docker-compose.yml.
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("tempo:4318"), // Tempo's OTLP/HTTP port
		otlptracehttp.WithInsecure(),             // Use http instead of https
	)
	if err != nil {
		return nil, err
	}

	// Create a new resource to identify this application
	// "go-api-service" will show up in Grafana
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("go-api-service"),
			semconv.ServiceVersion("1.0.0"),
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
