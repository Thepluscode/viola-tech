package middleware

import (
	"context"
	"math/rand"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/viola/gateway-api/internal/audit"
	"github.com/viola/gateway-api/internal/auth"
	auditv1 "github.com/viola/shared/proto/audit"
)

// claimsContextKey is the string key used to store/retrieve JWT claims from context.
// Must match the key used by auth.GetClaims.
const claimsContextKey = "claims"

// WithClaims adds claims to context
func WithClaims(ctx context.Context, c *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, c)
}

// ClaimsFrom extracts claims from context
func ClaimsFrom(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(claimsContextKey).(*auth.Claims)
	return c, ok
}

// AuthMiddleware validates Bearer tokens via OIDC
type AuthMiddleware struct {
	RequireBearer bool
	Verifier      *auth.Verifier

	// Optional: Audit failed auth attempts (sampled to avoid spam)
	Auditor     *audit.Emitter
	AuditSample float64 // 0.0-1.0, e.g., 0.1 = 10% sampling
}

// Handler returns the HTTP middleware handler
func (a AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.RequireBearer {
			next.ServeHTTP(w, r)
			return
		}

		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			a.emitFailedAuthAudit(r, "missing_token", "No Authorization header provided")
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimSpace(h[len("Bearer "):])

		c, err := a.Verifier.Verify(r.Context(), token)
		if err != nil {
			reason := "invalid_token"
			if strings.Contains(err.Error(), "expired") {
				reason = "token_expired"
			} else if strings.Contains(err.Error(), "issuer") {
				reason = "issuer_mismatch"
			} else if strings.Contains(err.Error(), "audience") {
				reason = "audience_mismatch"
			} else if strings.Contains(err.Error(), "kid") {
				reason = "unknown_key_id"
			}

			a.emitFailedAuthAudit(r, reason, err.Error())
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), c)))
	})
}

func (a *AuthMiddleware) emitFailedAuthAudit(r *http.Request, reason, details string) {
	if a.Auditor == nil {
		return
	}

	// Sample failed auth attempts to avoid overwhelming audit log
	if a.AuditSample > 0 && rand.Float64() > a.AuditSample {
		return // Not in sample
	}

	requestID := middleware.GetReqID(r.Context())

	ev := &auditv1.AuditEvent{
		ActorType:    "user",
		ActorId:      "unknown", // Can't extract from invalid token
		ActorIp:      r.RemoteAddr,
		ResourceType: "auth",
		ResourceId:   r.URL.Path,
		Action:       "authenticate",
		Outcome:      "denied",
		Reason:       reason,
		Metadata: map[string]string{
			"method":  r.Method,
			"path":    r.URL.Path,
			"details": details,
		},
	}

	// Use "system" as tenant ID for failed auth
	_ = a.Auditor.Emit(r.Context(), "system", requestID, ev)
}
