package observability

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/photosync/server/observability"

// HTTPMetrics holds HTTP-related metrics
type HTTPMetrics struct {
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	requestSize     metric.Int64Histogram
	responseSize    metric.Int64Histogram
	activeRequests  metric.Int64UpDownCounter
}

// NewHTTPMetrics creates HTTP metrics instruments
func NewHTTPMetrics() (*HTTPMetrics, error) {
	meter := otel.Meter(instrumentationName)

	requestCounter, err := meter.Int64Counter(
		"http.server.request_count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	requestSize, err := meter.Int64Histogram(
		"http.server.request.size",
		metric.WithDescription("HTTP request body size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	responseSize, err := meter.Int64Histogram(
		"http.server.response.size",
		metric.WithDescription("HTTP response body size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, err
	}

	return &HTTPMetrics{
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
		activeRequests:  activeRequests,
	}, nil
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// Flush implements http.Flusher
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// TracingMiddleware creates middleware that adds distributed tracing to HTTP requests
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(instrumentationName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming request
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Create span for this request
			spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.target", r.URL.Path),
					attribute.String("http.host", r.Host),
					attribute.String("http.scheme", getScheme(r)),
					attribute.String("http.user_agent", r.UserAgent()),
					attribute.String("net.peer.ip", r.RemoteAddr),
					attribute.String("http.route", r.URL.Path),
				),
			)
			defer span.End()

			// Wrap response writer to capture status code
			rw := newResponseWriter(w)

			// Add trace context to response headers
			propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))

			// Process request
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record response attributes
			span.SetAttributes(
				attribute.Int("http.status_code", rw.statusCode),
				attribute.Int64("http.response_content_length", rw.size),
			)

			// Set span status based on HTTP status code
			if rw.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(rw.statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

// MetricsMiddleware creates middleware that records HTTP metrics
func MetricsMiddleware(metrics *HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Common attributes for all metrics
			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
			}

			// Track active requests
			metrics.activeRequests.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			defer metrics.activeRequests.Add(r.Context(), -1, metric.WithAttributes(attrs...))

			// Record request size
			if r.ContentLength > 0 {
				metrics.requestSize.Record(r.Context(), r.ContentLength, metric.WithAttributes(attrs...))
			}

			// Wrap response writer
			rw := newResponseWriter(w)

			// Process request
			next.ServeHTTP(rw, r)

			// Add status code to attributes
			attrs = append(attrs, attribute.Int("http.status_code", rw.statusCode))

			// Record metrics
			duration := float64(time.Since(start).Milliseconds())
			metrics.requestDuration.Record(r.Context(), duration, metric.WithAttributes(attrs...))
			metrics.requestCounter.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			metrics.responseSize.Record(r.Context(), rw.size, metric.WithAttributes(attrs...))
		})
	}
}

func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}
