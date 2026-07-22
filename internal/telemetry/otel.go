package telemetry

import (
	"context"
	"fmt"
	"tkngate/internal/config"
	"tkngate/internal/logging"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var TracerProvider *sdktrace.TracerProvider

// InitOTEL sets up the OpenTelemetry TracerProvider and registers it globally.
func InitOTEL(ctx context.Context) error {
	if !config.Cfg.OTEL.Enabled {
		return nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.Cfg.OTEL.ServiceName),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter sdktrace.SpanExporter
	var endpoint string
	if config.Cfg.OTEL.ExporterType == "http" || config.Cfg.OTEL.ExporterType == "" {
		endpoint = config.Cfg.OTEL.Endpoint
		if endpoint == "" {
			endpoint = "localhost:4318" // Standard OTLP HTTP port
		}
		
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(), // typically internal to the k8s cluster
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported exporter type: %s", config.Cfg.OTEL.ExporterType)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	TracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	
	otel.SetTracerProvider(TracerProvider)
	
	logging.Logger.Info("OpenTelemetry initialized", "endpoint", endpoint)
	return nil
}
