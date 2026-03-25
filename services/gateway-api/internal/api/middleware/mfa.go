package middleware

import (
	"net/http"
	"strings"
)

// MFAMiddleware enforces MFA for sensitive operations
// Checks the `amr` (Authentication Methods Reference) claim for MFA indicators
type MFAMiddleware struct {
	required bool

	// MFA indicators to look for in the amr claim
	// Common values: "mfa", "otp", "sms", "totp", "duo", "fido", "phone"
	mfaIndicators map[string]bool
}

// NewMFAMiddleware creates a new MFA enforcement middleware
func NewMFAMiddleware(required bool, indicators []string) *MFAMiddleware {
	if len(indicators) == 0 {
		// Default MFA indicators
		indicators = []string{"mfa", "otp", "totp", "sms", "duo", "fido", "phone", "hwk"}
	}

	mfaMap := make(map[string]bool)
	for _, indicator := range indicators {
		mfaMap[strings.ToLower(indicator)] = true
	}

	return &MFAMiddleware{
		required:      required,
		mfaIndicators: mfaMap,
	}
}

// Handler returns the HTTP middleware handler
func (m *MFAMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.required {
			next.ServeHTTP(w, r)
			return
		}

		// Extract claims from context
		claims, ok := ClaimsFrom(r.Context())
		if !ok {
			// No claims = unauthenticated (should be caught by authn middleware)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Check amr claim for MFA
		if !m.hasMFA(claims.Raw) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"mfa_required","message":"Multi-factor authentication is required for this operation"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireMFA returns a middleware that requires MFA for a specific route
// Usage: r.With(mfaMiddleware.RequireMFA()).Patch("/api/v1/incidents/{id}", handler)
func (m *MFAMiddleware) RequireMFA() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Force enable MFA for this route
		tempMiddleware := &MFAMiddleware{
			required:      true,
			mfaIndicators: m.mfaIndicators,
		}
		return tempMiddleware.Handler(next)
	}
}

func (m *MFAMiddleware) hasMFA(rawClaims map[string]any) bool {
	// Check amr claim (Authentication Methods Reference)
	// This is a list of authentication methods used
	amrClaim, ok := rawClaims["amr"]
	if !ok {
		return false
	}

	// amr can be a string array or a single string
	switch amr := amrClaim.(type) {
	case []interface{}:
		for _, method := range amr {
			if methodStr, ok := method.(string); ok {
				if m.isMFAMethod(methodStr) {
					return true
				}
			}
		}
	case []string:
		for _, method := range amr {
			if m.isMFAMethod(method) {
				return true
			}
		}
	case string:
		return m.isMFAMethod(amr)
	}

	// Fallback: Check acr claim (Authentication Context Class Reference)
	// Some IdPs use this instead of amr
	acrClaim, ok := rawClaims["acr"]
	if ok {
		if acrStr, ok := acrClaim.(string); ok {
			// Common ACR values for MFA:
			// - "http://schemas.microsoft.com/claims/multipleauthn" (Entra ID)
			// - "https://refeds.org/profile/mfa" (REFEDS MFA)
			// - "urn:oasis:names:tc:SAML:2.0:ac:classes:MultiFactor"
			return strings.Contains(strings.ToLower(acrStr), "mfa") ||
				strings.Contains(strings.ToLower(acrStr), "multifactor") ||
				strings.Contains(acrStr, "multipleauthn")
		}
	}

	return false
}

func (m *MFAMiddleware) isMFAMethod(method string) bool {
	return m.mfaIndicators[strings.ToLower(method)]
}
