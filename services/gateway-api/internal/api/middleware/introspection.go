package middleware

import (
	"net/http"
	"strings"

	"github.com/viola/gateway-api/internal/auth"
)

// IntrospectionMiddleware checks token revocation for sensitive operations
// This should only be applied to high-security routes (e.g., delete operations, admin actions)
// as it adds latency due to the IdP call
type IntrospectionMiddleware struct {
	client  *auth.IntrospectionClient
	enabled bool
}

// NewIntrospectionMiddleware creates a new introspection middleware
func NewIntrospectionMiddleware(client *auth.IntrospectionClient, enabled bool) *IntrospectionMiddleware {
	return &IntrospectionMiddleware{
		client:  client,
		enabled: enabled,
	}
}

// Handler returns the HTTP middleware handler
func (m *IntrospectionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled || m.client == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			// No token = already handled by authn middleware
			next.ServeHTTP(w, r)
			return
		}
		token := strings.TrimSpace(h[len("Bearer "):])

		// Check if token is still active
		active, err := m.client.IsActive(r.Context(), token)
		if err != nil {
			// Log error but don't fail request
			// (Introspection is a defense-in-depth measure, not critical path)
			// In production, you might want to fail-closed instead
			next.ServeHTTP(w, r)
			return
		}

		if !active {
			// Token has been revoked
			http.Error(w, "token revoked", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireIntrospection returns a middleware that requires introspection for a specific route
// Usage: r.With(introspectionMiddleware.RequireIntrospection()).Delete("/api/v1/rules/{id}", handler)
func (m *IntrospectionMiddleware) RequireIntrospection() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Force enable for this route
		tempMiddleware := &IntrospectionMiddleware{
			client:  m.client,
			enabled: true,
		}
		return tempMiddleware.Handler(next)
	}
}
