// Package token issues and validates JWTs signed with RS256.
package token

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Claims represents the payload of a Viola JWT.
type Claims struct {
	// Standard claims
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`

	// Viola-specific claims
	TenantID string   `json:"tid"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

// Issuer signs JWTs using an RSA private key.
type Issuer struct {
	priv     *rsa.PrivateKey
	kid      string
	issuer   string
	audience string
	ttl      time.Duration
}

// Config configures the token issuer.
type Config struct {
	PrivateKey *rsa.PrivateKey
	KID        string
	Issuer     string
	Audience   string
	TTL        time.Duration
}

// New creates a token issuer.
func New(cfg Config) *Issuer {
	if cfg.TTL == 0 {
		cfg.TTL = time.Hour
	}
	return &Issuer{
		priv:     cfg.PrivateKey,
		kid:      cfg.KID,
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		ttl:      cfg.TTL,
	}
}

// Issue mints a new JWT for the given subject, tenant and role.
func (s *Issuer) Issue(subject, tenantID, email, role string) (string, error) {
	now := time.Now().UTC()

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": s.kid,
	}
	roles := []string{}
	if role != "" {
		roles = []string{role}
	}
	payload := Claims{
		Subject:   subject,
		Issuer:    s.issuer,
		Audience:  s.audience,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(s.ttl).Unix(),
		TenantID:  tenantID,
		Email:     email,
		Roles:     roles,
	}

	hb, err := jsonBase64(header)
	if err != nil {
		return "", err
	}
	pb, err := jsonBase64(payload)
	if err != nil {
		return "", err
	}

	signing := hb + "." + pb
	digest := sha256.Sum256([]byte(signing))

	sig, err := rsa.SignPKCS1v15(rand.Reader, s.priv, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signing + "." + sigB64, nil
}

// jsonBase64 marshals v to JSON then base64url-encodes it (no padding).
func jsonBase64(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ParseUnverified extracts claims from a JWT without signature verification.
// Only used server-side for logging; the gateway-api verifies via JWKS.
func ParseUnverified(tok string) (*Claims, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed jwt: expected 3 parts, got %d", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}
	return &c, nil
}
