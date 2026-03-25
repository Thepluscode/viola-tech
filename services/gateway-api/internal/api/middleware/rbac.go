package middleware

import (
	"net/http"

	"github.com/viola/gateway-api/internal/policy"
)

// Authorizer maps claims to permissions and checks route policies
type Authorizer interface {
	PermissionsFor(claims any) map[policy.Permission]bool
	Match(r *http.Request) (policy.RoutePolicy, bool)
}

// RBACMiddleware enforces route-level RBAC via central policy table
type RBACMiddleware struct {
	Authz Authorizer
}

// Handler returns the HTTP middleware handler
func (m RBACMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFrom(r.Context())
		if !ok {
			http.Error(w, "missing auth context", http.StatusUnauthorized)
			return
		}

		rp, ok := m.Authz.Match(r)
		if !ok {
			// No policy defined for this route - default deny
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		perms := m.Authz.PermissionsFor(claims)

		// ALL-of: must have all required permissions
		for _, p := range rp.AllOf {
			if !perms[p] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		// ANY-of: must have at least one required permission
		if len(rp.AnyOf) > 0 {
			okAny := false
			for _, p := range rp.AnyOf {
				if perms[p] {
					okAny = true
					break
				}
			}
			if !okAny {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
