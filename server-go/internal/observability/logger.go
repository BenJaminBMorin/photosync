package observability

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LogLevel represents log severity
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is a structured logger with trace context support
type Logger struct {
	mu          sync.RWMutex
	stdLogger   *log.Logger
	minLevel    LogLevel
	fields      map[string]interface{}
	serviceName string
}

var defaultLogger *Logger
var loggerOnce sync.Once

// NewLogger creates a new structured logger
func NewLogger(serviceName string, minLevel LogLevel) *Logger {
	return &Logger{
		stdLogger:   log.New(os.Stdout, "", 0),
		minLevel:    minLevel,
		fields:      make(map[string]interface{}),
		serviceName: serviceName,
	}
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		serviceName := os.Getenv("SERVICE_NAME")
		if serviceName == "" {
			serviceName = "photosync-server"
		}

		levelStr := os.Getenv("LOG_LEVEL")
		level := LevelInfo
		switch strings.ToLower(levelStr) {
		case "debug":
			level = LevelDebug
		case "warn":
			level = LevelWarn
		case "error":
			level = LevelError
		}

		defaultLogger = NewLogger(serviceName, level)
	})
	return defaultLogger
}

// SetOutput sets the output destination for standard logs
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdLogger = log.New(w, "", 0)
}

// WithField returns a new logger with the field added
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		stdLogger:   l.stdLogger,
		minLevel:    l.minLevel,
		fields:      newFields,
		serviceName: l.serviceName,
	}
}

// WithFields returns a new logger with the fields added
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		stdLogger:   l.stdLogger,
		minLevel:    l.minLevel,
		fields:      newFields,
		serviceName: l.serviceName,
	}
}

// WithContext returns a new logger with trace context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return l.WithFields(map[string]interface{}{
			"trace_id": span.SpanContext().TraceID().String(),
			"span_id":  span.SpanContext().SpanID().String(),
		})
	}
	return l
}

// Debug logs at debug level
func (l *Logger) Debug(msg string) {
	l.log(LevelDebug, msg)
}

// Debugf logs at debug level with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LevelDebug, fmt.Sprintf(format, args...))
}

// Info logs at info level
func (l *Logger) Info(msg string) {
	l.log(LevelInfo, msg)
}

// Infof logs at info level with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(format, args...))
}

// Warn logs at warn level
func (l *Logger) Warn(msg string) {
	l.log(LevelWarn, msg)
}

// Warnf logs at warn level with formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, fmt.Sprintf(format, args...))
}

// Error logs at error level
func (l *Logger) Error(msg string) {
	l.log(LevelError, msg)
}

// Errorf logs at error level with formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(format, args...))
}

func (l *Logger) log(level LogLevel, msg string) {
	if level < l.minLevel {
		return
	}

	now := time.Now()

	// Get caller information
	_, file, line, _ := runtime.Caller(2)
	// Shorten file path
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}

	// Build log line for stdout
	l.mu.RLock()
	var fieldParts []string
	for k, v := range l.fields {
		fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
	}
	l.mu.RUnlock()

	fieldStr := ""
	if len(fieldParts) > 0 {
		fieldStr = " " + strings.Join(fieldParts, " ")
	}

	logLine := fmt.Sprintf("%s [%s] %s:%d %s%s",
		now.Format("2006/01/02 15:04:05"),
		level.String(),
		file,
		line,
		msg,
		fieldStr,
	)

	l.stdLogger.Println(logLine)
}

// Convenience functions for package-level logging

// Debug logs at debug level
func Debug(msg string) {
	GetLogger().Debug(msg)
}

// Debugf logs at debug level with formatting
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info logs at info level
func Info(msg string) {
	GetLogger().Info(msg)
}

// Infof logs at info level with formatting
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn logs at warn level
func Warn(msg string) {
	GetLogger().Warn(msg)
}

// Warnf logs at warn level with formatting
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error logs at error level
func Error(msg string) {
	GetLogger().Error(msg)
}

// Errorf logs at error level with formatting
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// WithField returns a logger with the field
func WithField(key string, value interface{}) *Logger {
	return GetLogger().WithField(key, value)
}

// WithFields returns a logger with the fields
func WithFields(fields map[string]interface{}) *Logger {
	return GetLogger().WithFields(fields)
}

// WithContext returns a logger with trace context
func WithContext(ctx context.Context) *Logger {
	return GetLogger().WithContext(ctx)
}

// Custom attribute helpers for common fields
func RequestID(id string) attribute.KeyValue {
	return attribute.String("request_id", id)
}

func UserID(id string) attribute.KeyValue {
	return attribute.String("user_id", id)
}

func PhotoID(id string) attribute.KeyValue {
	return attribute.String("photo_id", id)
}

func CollectionID(id string) attribute.KeyValue {
	return attribute.String("collection_id", id)
}

func DeviceID(id string) attribute.KeyValue {
	return attribute.String("device_id", id)
}

func Operation(op string) attribute.KeyValue {
	return attribute.String("operation", op)
}

func Duration(d time.Duration) attribute.KeyValue {
	return attribute.Int64("duration_ms", d.Milliseconds())
}
