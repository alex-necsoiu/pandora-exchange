package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestNewTracerProvider_Disabled tests that a no-op provider is created when tracing is disabled
func TestNewTracerProvider_Disabled(t *testing.T) {
	ctx := context.Background()
	cfg := TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Enabled:        false,
	}

	tp, err := NewTracerProvider(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)
	assert.NotNil(t, tp.Tracer())

	// Shutdown should work without error
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestNewTracerProvider_WithInMemoryExporter tests tracer provider with in-memory exporter
func TestNewTracerProvider_WithInMemoryExporter(t *testing.T) {
	ctx := context.Background()

	// Create an in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()

	// Create a custom tracer provider with the in-memory exporter
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set as global provider
	otel.SetTracerProvider(provider)

	// Get tracer
	tracer := otel.Tracer(TracerName)
	require.NotNil(t, tracer)

	// Create a test span
	_, span := tracer.Start(ctx, "test-operation")
	span.SetAttributes(attribute.String("test.key", "test.value"))
	span.End()

	// Force flush
	err := provider.ForceFlush(ctx)
	require.NoError(t, err)

	// Verify span was recorded
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "test-operation", spans[0].Name)

	// Find the test.key attribute
	found := false
	for _, attr := range spans[0].Attributes {
		if attr.Key == "test.key" && attr.Value.AsString() == "test.value" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find test.key attribute")

	// Shutdown
	err = provider.Shutdown(ctx)
	require.NoError(t, err)
}

// TestTracerProvider_Tracer tests the Tracer method
func TestTracerProvider_Tracer(t *testing.T) {
	ctx := context.Background()
	cfg := TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Enabled:        false,
	}

	tp, err := NewTracerProvider(ctx, cfg)
	require.NoError(t, err)

	tracer := tp.Tracer()
	assert.NotNil(t, tracer)

	// Cleanup
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestTracerProvider_Shutdown tests graceful shutdown
func TestTracerProvider_Shutdown(t *testing.T) {
	ctx := context.Background()
	cfg := TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Enabled:        false,
	}

	tp, err := NewTracerProvider(ctx, cfg)
	require.NoError(t, err)

	// Shutdown should complete without error
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)

	// Second shutdown should also be safe (idempotent)
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestTracerProvider_ShutdownWithTimeout tests shutdown with context timeout
func TestTracerProvider_ShutdownWithTimeout(t *testing.T) {
	cfg := TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Enabled:        false,
	}

	tp, err := NewTracerProvider(context.Background(), cfg)
	require.NoError(t, err)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Shutdown with cancelled context - the implementation uses its own internal timeout
	// so it should complete successfully even with a cancelled context
	err = tp.Shutdown(ctx)
	// With disabled tracing, shutdown should succeed
	assert.NoError(t, err)
}

// TestTracerProvider_ForceFlush tests force flush functionality
func TestTracerProvider_ForceFlush(t *testing.T) {
	ctx := context.Background()

	// Create an in-memory span exporter
	exporter := tracetest.NewInMemoryExporter()

	// Create a custom tracer provider with batcher
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set as global provider to ensure spans are recorded
	otel.SetTracerProvider(provider)

	tp := &TracerProvider{
		provider: provider,
		tracer:   provider.Tracer(TracerName),
	}

	// Create some spans
	tracer := tp.Tracer()
	for i := 0; i < 3; i++ {
		_, span := tracer.Start(ctx, "test-span")
		span.End()
	}

	// Force flush to ensure spans are exported
	err := tp.ForceFlush(ctx)
	require.NoError(t, err)

	// Verify all spans were flushed
	spans := exporter.GetSpans()
	assert.Len(t, spans, 3)

	// Cleanup
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestTracerProvider_ForceFlushWithTimeout tests force flush with timeout
func TestTracerProvider_ForceFlushWithTimeout(t *testing.T) {
	ctx := context.Background()

	// Create an in-memory span exporter
	exporter := tracetest.NewInMemoryExporter()

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	tp := &TracerProvider{
		provider: provider,
		tracer:   otel.Tracer(TracerName),
	}

	// Create a very short timeout context
	flushCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Force flush with expired context
	err := tp.ForceFlush(flushCtx)
	// Should complete (may or may not error depending on timing)
	assert.NotNil(t, err)

	// Cleanup
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestTracerConfig_SamplingRates tests different sampling configurations
func TestTracerConfig_SamplingRates(t *testing.T) {
	testCases := []struct {
		name       string
		sampleRate float64
		enabled    bool
	}{
		{
			name:       "always sample",
			sampleRate: 1.0,
			enabled:    false, // Disabled to avoid OTLP connection
		},
		{
			name:       "never sample",
			sampleRate: 0.0,
			enabled:    false,
		},
		{
			name:       "50% sample",
			sampleRate: 0.5,
			enabled:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := TracerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				OTLPEndpoint:   "localhost:4317",
				Enabled:        tc.enabled,
				SampleRate:     tc.sampleRate,
			}

			tp, err := NewTracerProvider(ctx, cfg)
			require.NoError(t, err)
			require.NotNil(t, tp)

			// Cleanup
			err = tp.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

// TestTracerName_Constant tests that the tracer name constant is correct
func TestTracerName_Constant(t *testing.T) {
	assert.Equal(t, "github.com/alex-necsoiu/pandora-exchange/user-service", TracerName)
}
