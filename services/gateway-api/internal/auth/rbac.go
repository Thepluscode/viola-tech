package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RBAC struct {
	pool *pgxpool.Pool
}

func NewRBAC(pool *pgxpool.Pool) *RBAC {
	return &RBAC{pool: pool}
}

// RequirePermission checks if the user has permission to perform an action on a resource
func (rb *RBAC) RequirePermission(resource string, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := GetClaims(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if user has permission
			allowed, err := rb.CheckPermission(r.Context(), claims.TenantID, claims.Roles, resource, action)
			if err != nil {
				http.Error(w, "Internal error checking permissions", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, fmt.Sprintf("Forbidden: insufficient permissions for %s:%s", resource, action), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CheckPermission checks if any of the user's roles have permission
func (rb *RBAC) CheckPermission(ctx context.Context, tenantID string, roles []string, resource string, action string) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}

	// Check tenant-specific policies first, then fall back to global (*) policies
	query := `
		SELECT allowed
		FROM rbac_policies
		WHERE (tenant_id = $1 OR tenant_id = '*')
			AND role = ANY($2)
			AND resource = $3
			AND action = $4
		ORDER BY
			CASE WHEN tenant_id = $1 THEN 0 ELSE 1 END,  -- tenant-specific first
			allowed DESC                                  -- allowed=true first
		LIMIT 1
	`

	var allowed bool
	err := rb.pool.QueryRow(ctx, query, tenantID, roles, resource, action).Scan(&allowed)
	if err != nil {
		// If no policy found, default to deny
		return false, nil
	}

	return allowed, nil
}
