package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/viola/shared/observability/logging"
	"github.com/viola/shared/observability/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MetricsMiddleware tracks HTTP request metrics
func MetricsMiddleware(m *metrics.Registry) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Track request
			defer func() {
				duration := time.Since(start).Seconds()
				endpoint := r.URL.Path
				method := r.Method
				status := strconv.Itoa(ww.Status())

				// Track metrics
				m.HTTPRequests.WithLabelValues(endpoint, method, status).Inc()
				m.HTTPDuration.WithLabelValues(endpoint, method).Observe(duration)

				// Track errors (4xx, 5xx)
				if ww.Status() >= 400 {
					m.HTTPErrors.WithLabelValues(endpoint, method, status).Inc()
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// LoggingMiddleware adds structured logging to HTTP requests
func LoggingMiddleware(logger *logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Extract or generate request ID
			requestID := middleware.GetReqID(r.Context())
			if requestID == "" {
				requestID = strconv.FormatUint(middleware.NextRequestID(), 10)
			}

			// Add request ID to context
			ctx := logging.WithContext(r.Context(), requestID, "")

			// Log request
			logger.Info(ctx, "HTTP request", map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
				"remote": r.RemoteAddr,
			})

			// Process request
			next.ServeHTTP(ww, r.WithContext(ctx))

			// Log response
			duration := time.Since(start)
			logger.Info(ctx, "HTTP response", map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      ww.Status(),
				"duration_ms": duration.Milliseconds(),
				"bytes":       ww.BytesWritten(),
			})
		})
	}
}

// TracingMiddleware adds OpenTelemetry tracing to HTTP requests
func TracingMiddleware(tracer trace.Tracer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			defer span.End()

			// Add request attributes
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.scheme", r.URL.Scheme),
				attribute.String("http.host", r.Host),
				attribute.String("http.user_agent", r.UserAgent()),
			)

			// Capture response status
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(ctx))

			// Add response attributes
			span.SetAttributes(
				attribute.Int("http.status_code", ww.Status()),
				attribute.Int("http.response_size", ww.BytesWritten()),
			)

			// Mark span as error if status >= 400
			if ww.Status() >= 400 {
				span.SetAttributes(attribute.Bool("error", true))
			}
		})
	}
}
