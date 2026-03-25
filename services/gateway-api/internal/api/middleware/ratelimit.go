package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/viola/gateway-api/internal/auth"
	"github.com/viola/gateway-api/internal/ratelimit"
)

// RateLimitMiddleware enforces per-user rate limiting
type RateLimitMiddleware struct {
	limiter *ratelimit.Limiter
	enabled bool

	// Claim fields to use for key construction
	keyClaims []string // e.g., ["sub", "tid"]

	// Custom limit per-user claim (optional)
	customLimitClaim string // e.g., "viola_rl_per_min"
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	Enabled          bool
	MaxRequests      int      // Default requests per window
	WindowMinutes    int      // Window size in minutes
	KeyClaims        []string // Claims to use for key (e.g., ["sub", "tid"])
	CustomLimitClaim string   // Optional claim for per-user limits
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(cfg RateLimitConfig) *RateLimitMiddleware {
	if !cfg.Enabled {
		return &RateLimitMiddleware{enabled: false}
	}

	if len(cfg.KeyClaims) == 0 {
		cfg.KeyClaims = []string{"sub", "tid"} // Default: user + tenant
	}

	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 120 // Default: 120 requests
	}

	if cfg.WindowMinutes == 0 {
		cfg.WindowMinutes = 1 // Default: 1 minute
	}

	limiter := ratelimit.New(ratelimit.Config{
		WindowSize:  time.Duration(cfg.WindowMinutes) * time.Minute,
		MaxRequests: cfg.MaxRequests,
	})

	return &RateLimitMiddleware{
		limiter:          limiter,
		enabled:          true,
		keyClaims:        cfg.KeyClaims,
		customLimitClaim: cfg.CustomLimitClaim,
	}
}

// Handler returns the HTTP middleware handler
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Extract claims from context
		claims, ok := ClaimsFrom(r.Context())
		if !ok {
			// No claims = unauthenticated request
			// Rate limit by IP address as fallback
			key := "ip:" + r.RemoteAddr
			if !m.limiter.Allow(key) {
				m.writeTooManyRequests(w, key)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Build key from claims
		key := m.buildKey(claims)

		// Check custom per-user limit (if configured)
		customLimit := m.getCustomLimit(claims)
		if customLimit > 0 {
			// TODO: Support per-user limits (requires separate limiter instance)
			// For now, use the default limiter
		}

		// Check rate limit
		if !m.limiter.Allow(key) {
			m.writeTooManyRequests(w, key)
			return
		}

		// Add rate limit headers
		remaining := m.limiter.Remaining(key)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(m.limiter.Limit()))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

		next.ServeHTTP(w, r)
	})
}

func (m *RateLimitMiddleware) buildKey(claims *auth.Claims) string {
	parts := make([]string, 0, len(m.keyClaims))

	for _, claimKey := range m.keyClaims {
		var value string
		switch claimKey {
		case "sub", "subject":
			value = claims.Subject
		case "tid", "tenant_id":
			value = claims.TenantID
		case "email":
			value = claims.Email
		default:
			// Try to get from raw claims
			if v, ok := claims.Raw[claimKey]; ok {
				if str, ok := v.(string); ok {
					value = str
				}
			}
		}

		if value != "" {
			parts = append(parts, value)
		}
	}

	// Join with colon separator
	key := ""
	for i, part := range parts {
		if i > 0 {
			key += ":"
		}
		key += part
	}

	return key
}

func (m *RateLimitMiddleware) getCustomLimit(claims *auth.Claims) int {
	if m.customLimitClaim == "" {
		return 0
	}

	// Try to extract custom limit from claims
	if v, ok := claims.Raw[m.customLimitClaim]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case string:
			if limit, err := strconv.Atoi(val); err == nil {
				return limit
			}
		}
	}

	return 0
}

func (m *RateLimitMiddleware) writeTooManyRequests(w http.ResponseWriter, key string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(m.limiter.Limit()))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.WriteHeader(http.StatusTooManyRequests)

	response := fmt.Sprintf(`{"error":"rate_limit_exceeded","message":"Too many requests. Please try again later.","key":"%s"}`, key)
	w.Write([]byte(response))
}
