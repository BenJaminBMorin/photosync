package observability

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Tracer returns a tracer for the given name
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span from context
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(instrumentationName).Start(ctx, name, opts...)
}

// StartDBSpan starts a span for database operations
func StartDBSpan(ctx context.Context, operation, table string) (context.Context, trace.Span) {
	return StartSpan(ctx, fmt.Sprintf("DB %s %s", operation, table),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", operation),
			attribute.String("db.sql.table", table),
		),
	)
}

// StartServiceSpan starts a span for service operations
func StartServiceSpan(ctx context.Context, service, operation string) (context.Context, trace.Span) {
	return StartSpan(ctx, fmt.Sprintf("%s.%s", service, operation),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("service.component", service),
			attribute.String("service.operation", operation),
		),
	)
}

// RecordError records an error on the span
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSuccess marks the span as successful
func SetSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// AddEvent adds an event to the span
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// DatabaseMetrics holds database-related metrics
type DatabaseMetrics struct {
	queryDuration   metric.Float64Histogram
	queryCount      metric.Int64Counter
	errorCount      metric.Int64Counter
	connectionCount metric.Int64UpDownCounter
}

// NewDatabaseMetrics creates database metrics instruments
func NewDatabaseMetrics() (*DatabaseMetrics, error) {
	meter := otel.Meter(instrumentationName)

	queryDuration, err := meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Database query duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	queryCount, err := meter.Int64Counter(
		"db.query.count",
		metric.WithDescription("Total number of database queries"),
		metric.WithUnit("{queries}"),
	)
	if err != nil {
		return nil, err
	}

	errorCount, err := meter.Int64Counter(
		"db.error.count",
		metric.WithDescription("Total number of database errors"),
		metric.WithUnit("{errors}"),
	)
	if err != nil {
		return nil, err
	}

	connectionCount, err := meter.Int64UpDownCounter(
		"db.connection.count",
		metric.WithDescription("Number of active database connections"),
		metric.WithUnit("{connections}"),
	)
	if err != nil {
		return nil, err
	}

	return &DatabaseMetrics{
		queryDuration:   queryDuration,
		queryCount:      queryCount,
		errorCount:      errorCount,
		connectionCount: connectionCount,
	}, nil
}

// RecordQuery records a database query metrics
func (m *DatabaseMetrics) RecordQuery(ctx context.Context, operation, table string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("db.operation", operation),
		attribute.String("db.sql.table", table),
	}

	m.queryCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.queryDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))

	if err != nil {
		m.errorCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// TraceDB wraps sql.DB with tracing
type TraceDB struct {
	db      *sql.DB
	metrics *DatabaseMetrics
}

// NewTraceDB creates a traced database wrapper
func NewTraceDB(db *sql.DB) (*TraceDB, error) {
	metrics, err := NewDatabaseMetrics()
	if err != nil {
		return nil, err
	}

	return &TraceDB{
		db:      db,
		metrics: metrics,
	}, nil
}

// QueryContext executes a query with tracing
func (t *TraceDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := StartSpan(ctx, "DB Query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.statement", truncateQuery(query)),
		),
	)
	defer span.End()

	start := time.Now()
	rows, err := t.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		RecordError(span, err)
	} else {
		SetSuccess(span)
	}

	span.SetAttributes(attribute.Int64("db.query_duration_ms", duration.Milliseconds()))

	return rows, err
}

// ExecContext executes a statement with tracing
func (t *TraceDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := StartSpan(ctx, "DB Exec",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.statement", truncateQuery(query)),
		),
	)
	defer span.End()

	start := time.Now()
	result, err := t.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		RecordError(span, err)
	} else {
		SetSuccess(span)
		if rowsAffected, raErr := result.RowsAffected(); raErr == nil {
			span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))
		}
	}

	span.SetAttributes(attribute.Int64("db.query_duration_ms", duration.Milliseconds()))

	return result, err
}

// QueryRowContext executes a query that returns a single row with tracing
func (t *TraceDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, span := StartSpan(ctx, "DB QueryRow",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.statement", truncateQuery(query)),
		),
	)
	// Note: span.End() should be called after scanning the row
	// This is a limitation of the sql.Row interface

	row := t.db.QueryRowContext(ctx, query, args...)
	span.End()
	return row
}

// DB returns the underlying database connection
func (t *TraceDB) DB() *sql.DB {
	return t.db
}

func truncateQuery(query string) string {
	if len(query) > 500 {
		return query[:500] + "..."
	}
	return query
}

// BusinessMetrics holds custom business metrics
type BusinessMetrics struct {
	photoUploads     metric.Int64Counter
	photoDownloads   metric.Int64Counter
	syncOperations   metric.Int64Counter
	authAttempts     metric.Int64Counter
	storageUsed      metric.Int64UpDownCounter
	activeUsers      metric.Int64UpDownCounter
}

// NewBusinessMetrics creates business metrics instruments
func NewBusinessMetrics() (*BusinessMetrics, error) {
	meter := otel.Meter(instrumentationName)

	photoUploads, err := meter.Int64Counter(
		"photosync.photo.uploads",
		metric.WithDescription("Total number of photo uploads"),
		metric.WithUnit("{uploads}"),
	)
	if err != nil {
		return nil, err
	}

	photoDownloads, err := meter.Int64Counter(
		"photosync.photo.downloads",
		metric.WithDescription("Total number of photo downloads"),
		metric.WithUnit("{downloads}"),
	)
	if err != nil {
		return nil, err
	}

	syncOperations, err := meter.Int64Counter(
		"photosync.sync.operations",
		metric.WithDescription("Total number of sync operations"),
		metric.WithUnit("{operations}"),
	)
	if err != nil {
		return nil, err
	}

	authAttempts, err := meter.Int64Counter(
		"photosync.auth.attempts",
		metric.WithDescription("Total number of authentication attempts"),
		metric.WithUnit("{attempts}"),
	)
	if err != nil {
		return nil, err
	}

	storageUsed, err := meter.Int64UpDownCounter(
		"photosync.storage.bytes",
		metric.WithDescription("Storage used in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	activeUsers, err := meter.Int64UpDownCounter(
		"photosync.users.active",
		metric.WithDescription("Number of active users"),
		metric.WithUnit("{users}"),
	)
	if err != nil {
		return nil, err
	}

	return &BusinessMetrics{
		photoUploads:   photoUploads,
		photoDownloads: photoDownloads,
		syncOperations: syncOperations,
		authAttempts:   authAttempts,
		storageUsed:    storageUsed,
		activeUsers:    activeUsers,
	}, nil
}

// RecordPhotoUpload records a photo upload
func (m *BusinessMetrics) RecordPhotoUpload(ctx context.Context, userID string, fileSize int64, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("user_id", userID),
		attribute.Bool("success", success),
	}
	m.photoUploads.Add(ctx, 1, metric.WithAttributes(attrs...))
	if success {
		m.storageUsed.Add(ctx, fileSize)
	}
}

// RecordPhotoDownload records a photo download
func (m *BusinessMetrics) RecordPhotoDownload(ctx context.Context, userID string) {
	attrs := []attribute.KeyValue{
		attribute.String("user_id", userID),
	}
	m.photoDownloads.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordSyncOperation records a sync operation
func (m *BusinessMetrics) RecordSyncOperation(ctx context.Context, userID, operationType string, photoCount int) {
	attrs := []attribute.KeyValue{
		attribute.String("user_id", userID),
		attribute.String("operation_type", operationType),
		attribute.Int("photo_count", photoCount),
	}
	m.syncOperations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordAuthAttempt records an authentication attempt
func (m *BusinessMetrics) RecordAuthAttempt(ctx context.Context, method string, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("auth_method", method),
		attribute.Bool("success", success),
	}
	m.authAttempts.Add(ctx, 1, metric.WithAttributes(attrs...))
}
