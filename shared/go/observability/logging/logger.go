package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel defines the severity of a log message
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

// ContextKey for storing request metadata in context
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	TenantIDKey  ContextKey = "tenant_id"
)

// Logger provides structured logging with context propagation
type Logger struct {
	service string
	level   LogLevel
	output  *log.Logger
}

// LogEntry represents a structured log message
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	RequestID string                 `json:"request_id,omitempty"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// New creates a new structured logger for a service
func New(serviceName string, level LogLevel) *Logger {
	return &Logger{
		service: serviceName,
		level:   level,
		output:  log.New(os.Stdout, "", 0),
	}
}

// WithContext creates a context with request metadata
func WithContext(ctx context.Context, requestID, tenantID string) context.Context {
	ctx = context.WithValue(ctx, RequestIDKey, requestID)
	if tenantID != "" {
		ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	}
	return ctx
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(TenantIDKey).(string); ok {
		return id
	}
	return ""
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.shouldLog(DEBUG) {
		l.log(ctx, DEBUG, msg, fields)
	}
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.shouldLog(INFO) {
		l.log(ctx, INFO, msg, fields)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.shouldLog(WARN) {
		l.log(ctx, WARN, msg, fields)
	}
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.shouldLog(ERROR) {
		l.log(ctx, ERROR, msg, fields)
	}
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if l.shouldLog(DEBUG) {
		l.log(ctx, DEBUG, fmt.Sprintf(format, args...), nil)
	}
}

// Infof logs a formatted info message
func (l *Logger) Infof(ctx context.Context, format string, args ...interface{}) {
	if l.shouldLog(INFO) {
		l.log(ctx, INFO, fmt.Sprintf(format, args...), nil)
	}
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if l.shouldLog(WARN) {
		l.log(ctx, WARN, fmt.Sprintf(format, args...), nil)
	}
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if l.shouldLog(ERROR) {
		l.log(ctx, ERROR, fmt.Sprintf(format, args...), nil)
	}
}

func (l *Logger) log(ctx context.Context, level LogLevel, msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     string(level),
		Service:   l.service,
		RequestID: GetRequestID(ctx),
		TenantID:  GetTenantID(ctx),
		Message:   msg,
		Fields:    fields,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON encoding fails
		l.output.Printf("[%s] %s: %s (JSON error: %v)", level, l.service, msg, err)
		return
	}

	l.output.Println(string(data))
}

func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
	}

	return levels[level] >= levels[l.level]
}

// ParseLevel converts a string to a LogLevel
func ParseLevel(s string) LogLevel {
	switch s {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}
