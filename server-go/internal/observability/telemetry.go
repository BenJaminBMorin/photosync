package observability

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds telemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	Enabled        bool
}

// Telemetry holds the telemetry providers
type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	config         Config
}

// NewConfig creates telemetry config from environment variables
func NewConfig(serviceName, serviceVersion string) Config {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	enabled := os.Getenv("OTEL_ENABLED")
	if enabled == "" {
		enabled = "true" // Enabled by default
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	return Config{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    env,
		OTLPEndpoint:   endpoint,
		Enabled:        enabled == "true" || enabled == "1",
	}
}

// Initialize sets up OpenTelemetry with tracing and metrics
func Initialize(ctx context.Context, cfg Config) (*Telemetry, error) {
	if !cfg.Enabled {
		log.Println("Telemetry disabled (set OTEL_ENABLED=true to enable)")
		return &Telemetry{config: cfg}, nil
	}

	log.Printf("Initializing telemetry with endpoint: %s", cfg.OTLPEndpoint)

	// Create resource with service information
	// Use empty schema URL to avoid conflicts with SDK's default schema version
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
			attribute.String("deployment.environment", cfg.Environment),
		),
		resource.WithHost(),
	)
	if err != nil {
		return nil, err
	}

	// Initialize trace provider
	tracerProvider, err := initTracer(ctx, cfg.OTLPEndpoint, res)
	if err != nil {
		log.Printf("Warning: Failed to initialize tracer: %v", err)
		// Continue without tracing
	} else {
		otel.SetTracerProvider(tracerProvider)
	}

	// Initialize meter provider
	meterProvider, err := initMeter(ctx, cfg.OTLPEndpoint, res)
	if err != nil {
		log.Printf("Warning: Failed to initialize meter: %v", err)
		// Continue without metrics
	} else {
		otel.SetMeterProvider(meterProvider)
	}

	// Set up propagators for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("Telemetry initialized successfully")

	return &Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		config:         cfg,
	}, nil
}

func initTracer(ctx context.Context, endpoint string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	return tp, nil
}

func initMeter(ctx context.Context, endpoint string, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(30*time.Second),
		)),
		sdkmetric.WithResource(res),
	)

	return mp, nil
}

// Shutdown gracefully shuts down telemetry providers
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if !t.config.Enabled {
		return nil
	}

	log.Println("Shutting down telemetry...")

	var errs []error

	if t.TracerProvider != nil {
		if err := t.TracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if t.MeterProvider != nil {
		if err := t.MeterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}
