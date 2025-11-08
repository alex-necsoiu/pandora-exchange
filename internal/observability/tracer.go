// Package observability provides OpenTelemetry tracing setup for the User Service.
package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the instrumentation name for this service
	TracerName = "github.com/alex-necsoiu/pandora-exchange/user-service"
)

// TracerConfig holds OpenTelemetry tracer configuration
type TracerConfig struct {
	// ServiceName is the name of the service
	ServiceName string
	// ServiceVersion is the version of the service
	ServiceVersion string
	// Environment is the deployment environment (dev, sandbox, audit, prod)
	Environment string
	// OTLPEndpoint is the OpenTelemetry collector endpoint (e.g., "localhost:4317")
	OTLPEndpoint string
	// Enabled controls whether tracing is enabled
	Enabled bool
	// SampleRate is the sampling rate (0.0 to 1.0)
	SampleRate float64
}

// TracerProvider wraps the OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// NewTracerProvider creates and configures an OpenTelemetry tracer provider
// Returns error if OTLP exporter cannot be initialized
func NewTracerProvider(ctx context.Context, cfg TracerConfig) (*TracerProvider, error) {
	// If tracing is disabled, return a no-op provider
	if !cfg.Enabled {
		return &TracerProvider{
			provider: sdktrace.NewTracerProvider(),
			tracer:   otel.Tracer(TracerName),
		}, nil
	}

	// Create OTLP trace exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(), // Use TLS in production
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure sampling based on sample rate
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Set global propagator for context propagation (W3C Trace Context)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: provider,
		tracer:   otel.Tracer(TracerName),
	}, nil
}

// Tracer returns the configured tracer for creating spans
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// Shutdown gracefully shuts down the tracer provider
// Flushes any pending spans to the exporter
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tp.provider.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	return nil
}

// ForceFlush forces the tracer provider to flush all pending spans
func (tp *TracerProvider) ForceFlush(ctx context.Context) error {
	// Create a timeout context for flush
	flushCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := tp.provider.ForceFlush(flushCtx); err != nil {
		return fmt.Errorf("failed to flush tracer provider: %w", err)
	}

	return nil
}
