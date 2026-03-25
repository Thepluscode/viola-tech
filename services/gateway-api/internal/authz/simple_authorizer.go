package authz

import (
	"net/http"
	"strings"

	"github.com/viola/gateway-api/internal/auth"
	"github.com/viola/gateway-api/internal/policy"
)

// SimpleAuthorizer maps JWT claims to permissions
type SimpleAuthorizer struct{}

// PermissionsFor converts claims into a set of permissions
func (a SimpleAuthorizer) PermissionsFor(c any) map[policy.Permission]bool {
	claims := c.(*auth.Claims)
	out := map[policy.Permission]bool{}

	// Admin shortcut
	for _, r := range claims.Roles {
		if strings.EqualFold(r, "ViolaAdmin") || strings.EqualFold(r, "Admin") {
			out[policy.PermAdmin] = true
		}
	}

	// Scope mapping (recommended for APIs)
	// Scopes are fine-grained permissions issued by the IdP
	for _, s := range claims.Scopes {
		switch s {
		case "incidents.read":
			out[policy.PermIncidentsRead] = true
		case "incidents.write":
			out[policy.PermIncidentsWrite] = true
		case "alerts.read":
			out[policy.PermAlertsRead] = true
		case "alerts.write":
			out[policy.PermAlertsWrite] = true
		case "rules.write":
			out[policy.PermRulesWrite] = true
		}
	}

	// Role mapping (fallback for coarse-grained roles)
	for _, r := range claims.Roles {
		switch r {
		case "SOCReader":
			out[policy.PermIncidentsRead] = true
			out[policy.PermAlertsRead] = true
		case "SOCResponder":
			out[policy.PermIncidentsRead] = true
			out[policy.PermIncidentsWrite] = true
			out[policy.PermAlertsRead] = true
			out[policy.PermAlertsWrite] = true
		case "SOCEngineer":
			out[policy.PermIncidentsRead] = true
			out[policy.PermIncidentsWrite] = true
			out[policy.PermAlertsRead] = true
			out[policy.PermAlertsWrite] = true
			out[policy.PermRulesWrite] = true
		}
	}

	return out
}

// Match finds the policy for a given HTTP request
// Matches method + path pattern (supports {id} placeholders)
func (a SimpleAuthorizer) Match(r *http.Request) (policy.RoutePolicy, bool) {
	path := r.URL.Path
	method := r.Method

	// Try exact match first
	for _, p := range policy.Policies {
		if p.Method != method {
			continue
		}
		if p.Path == path {
			return p, true
		}
	}

	// Try pattern match with {id} placeholder
	for _, p := range policy.Policies {
		if p.Method != method {
			continue
		}
		if matchPathPattern(p.Path, path) {
			return p, true
		}
	}

	return policy.RoutePolicy{}, false
}

// matchPathPattern matches patterns like "/api/v1/incidents/{id}" against "/api/v1/incidents/123"
func matchPathPattern(pattern, path string) bool {
	if !strings.Contains(pattern, "{") {
		return false
	}

	pParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(pParts) != len(pathParts) {
		return false
	}

	for i := range pParts {
		// Match placeholders like {id}, {alertId}, etc.
		if strings.HasPrefix(pParts[i], "{") && strings.HasSuffix(pParts[i], "}") {
			continue
		}
		// Literal segment must match exactly
		if pParts[i] != pathParts[i] {
			return false
		}
	}

	return true
}
