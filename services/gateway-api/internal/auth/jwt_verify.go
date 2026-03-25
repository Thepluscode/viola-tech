package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Verifier validates JWTs using JWKS
type Verifier struct {
	Issuer   string
	Audience string
	Algos    map[string]bool
	Skew     time.Duration
	JWKS     JWKSPublicKeyProvider
}

// Verify parses and validates a JWT token
func (v *Verifier) Verify(ctx context.Context, tokenString string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods(v.allowedAlgos()))

	tok, err := parser.Parse(tokenString, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid")
		}

		if pk, ok := v.JWKS.Get(ctx, kid); ok {
			return pk, nil
		}

		// Key rotation tolerance: force refresh then retry lookup once.
		_ = v.JWKS.Refresh(ctx)
		if pk, ok := v.JWKS.Get(ctx, kid); ok {
			return pk, nil
		}
		return nil, errors.New("unknown kid")
	})

	if err != nil || tok == nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}

	mapc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	// Validate issuer/audience/exp with skew
	if iss, _ := mapc["iss"].(string); v.Issuer != "" && iss != v.Issuer {
		return nil, errors.New("issuer mismatch")
	}
	if v.Audience != "" && !audContains(mapc["aud"], v.Audience) {
		return nil, errors.New("audience mismatch")
	}
	exp, err := parseNumericDate(mapc["exp"])
	if err != nil {
		return nil, errors.New("missing/invalid exp")
	}
	if time.Now().UTC().After(exp.Add(v.Skew)) {
		return nil, errors.New("token expired")
	}

	c := &Claims{
		Issuer:   v.Issuer,
		Audience: v.Audience,
		Expiry:   exp,
		Raw:      map[string]any(mapc),
	}

	// Common claim mapping (works for Entra + Okta-ish)
	c.Subject, _ = mapc["sub"].(string)
	c.Email, _ = mapc["email"].(string)
	if c.Email == "" {
		// Entra often uses preferred_username / upn
		c.Email, _ = mapc["preferred_username"].(string)
		if c.Email == "" {
			c.Email, _ = mapc["upn"].(string)
		}
	}

	// Tenant: prefer x-tenant-id header at API boundary, but JWT may contain tid (Entra)
	c.TenantID, _ = mapc["tid"].(string)
	if c.TenantID == "" {
		c.TenantID, _ = mapc["tenant_id"].(string)
	}

	c.Roles = stringArray(mapc["roles"])
	c.Scopes = strings.Fields(asString(mapc["scp"])) // Entra uses "scp": "a b c"
	if len(c.Scopes) == 0 {
		c.Scopes = stringArray(mapc["scope"]) // Okta sometimes
	}
	return c, nil
}

func (v *Verifier) allowedAlgos() []string {
	out := make([]string, 0, len(v.Algos))
	for a := range v.Algos {
		out = append(out, a)
	}
	return out
}

func audContains(a any, want string) bool {
	switch t := a.(type) {
	case string:
		return t == want
	case []any:
		for _, x := range t {
			if xs, ok := x.(string); ok && xs == want {
				return true
			}
		}
	}
	return false
}

func parseNumericDate(v any) (time.Time, error) {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0).UTC(), nil
	case int64:
		return time.Unix(t, 0).UTC(), nil
	default:
		return time.Time{}, errors.New("bad numeric date")
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringArray(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if xs, ok := x.(string); ok {
				out = append(out, xs)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}
