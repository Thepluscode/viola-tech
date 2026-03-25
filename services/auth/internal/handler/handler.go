// Package handler provides HTTP handlers for the auth service.
//
// Endpoints:
//
//	POST /token          Issue a JWT (dev/service-to-service use)
//	GET  /.well-known/jwks.json   Public keys for JWT verification
//	GET  /health         Liveness probe
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/viola/auth/internal/token"
)

// Handler holds dependencies for the HTTP handlers.
type Handler struct {
	issuer *token.Issuer
	jwks   []byte // pre-built JWKS JSON
}

// New creates a Handler.
func New(issuer *token.Issuer, jwks []byte) *Handler {
	return &Handler{issuer: issuer, jwks: jwks}
}

// TokenRequest is the JSON body for POST /token.
type TokenRequest struct {
	Subject  string `json:"sub"`
	TenantID string `json:"tid"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// TokenResponse is returned on success.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// Token handles POST /token — issues a JWT.
func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, "sub is required")
		return
	}
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tid is required")
		return
	}

	tok, err := h.issuer.Issue(req.Subject, req.TenantID, req.Email, req.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken: tok,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	})
}

// JWKS handles GET /.well-known/jwks.json — returns the public key set.
func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.WriteHeader(http.StatusOK)
	w.Write(h.jwks)
}

// Health handles GET /health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
