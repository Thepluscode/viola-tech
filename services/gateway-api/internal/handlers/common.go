package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/viola/gateway-api/internal/auth"
)

type ErrorResponse struct {
	Error  string `json:"error"`
	Code   string `json:"code,omitempty"`
	Status int    `json:"status"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string, code string) {
	writeJSON(w, status, ErrorResponse{
		Error:  message,
		Code:   code,
		Status: status,
	})
}

// getTenantID extracts the tenant ID from the validated JWT claims stored in
// the request context by AuthMiddleware. Returns "" if claims are absent.
func getTenantID(r *http.Request) string {
	claims, err := auth.GetClaims(r)
	if err != nil {
		return ""
	}
	return claims.TenantID
}

// toJSONString serialises v to a JSON string; returns "null" on error.
// Used to convert metadata values to strings for audit events (C2: consistent encoding).
func toJSONString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// requireTenantMatch is a defence-in-depth guard (C3).
// After loading a resource from the store, call this to verify that the
// resource's tenant_id matches the JWT-derived tenant ID. The store queries
// already include WHERE tenant_id=$1, so a mismatch here indicates a serious
// data bug and must be treated as a 403, not a 404, to avoid leaking IDs.
//
// Usage:
//
//	if !requireTenantMatch(w, jwtTenantID, resource.TenantID) { return }
func requireTenantMatch(w http.ResponseWriter, jwtTenantID, resourceTenantID string) bool {
	if jwtTenantID == resourceTenantID {
		return true
	}
	writeError(w, http.StatusForbidden, "Resource does not belong to your tenant", "FORBIDDEN")
	return false
}
