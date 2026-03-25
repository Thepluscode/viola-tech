package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig configures CORS middleware
type CORSConfig struct {
	AllowedOrigins   []string // e.g., ["https://viola.com", "https://app.viola.com"]
	AllowedMethods   []string // e.g., ["GET", "POST", "PATCH", "DELETE"]
	AllowedHeaders   []string // e.g., ["Authorization", "Content-Type"]
	ExposedHeaders   []string // e.g., ["X-Request-ID", "X-RateLimit-Remaining"]
	AllowCredentials bool
	MaxAge           int // Preflight cache duration in seconds
}

// DefaultCORSConfig returns a sensible default configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"}, // Override via CORS_ORIGINS env var in production
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"X-Request-ID",
			"X-Correlation-ID",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: false,
		MaxAge:           3600, // 1 hour
	}
}

// CORSMiddleware returns a CORS middleware handler
func CORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Handle preflight request
				if r.Method == http.MethodOptions {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))

					if len(cfg.ExposedHeaders) > 0 {
						w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
					}

					w.WriteHeader(http.StatusNoContent)
					return
				}

				// Set exposed headers for actual requests
				if len(cfg.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == "*" {
			return true
		}
		if o == origin {
			return true
		}
		// Support wildcards like https://*.viola.com
		if strings.HasPrefix(o, "*.") {
			domain := o[2:] // Remove "*."
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
