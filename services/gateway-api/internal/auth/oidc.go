package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OIDC handles OIDC token validation with JWKS
type OIDC struct {
	issuer      string
	audience    string
	jwksURL     string
	keys        map[string]*rsa.PublicKey
	keysMu      sync.RWMutex
	lastRefresh time.Time
}

type OIDCConfig struct {
	Issuer   string
	Audience string
	JWKSURL  string
}

func NewOIDC(cfg OIDCConfig) (*OIDC, error) {
	if cfg.Issuer == "" || cfg.Audience == "" || cfg.JWKSURL == "" {
		return nil, errors.New("auth: issuer, audience, and jwks_url required")
	}

	o := &OIDC{
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		jwksURL:  cfg.JWKSURL,
		keys:     make(map[string]*rsa.PublicKey),
	}

	// Initial JWKS fetch
	if err := o.refreshKeys(); err != nil {
		return nil, fmt.Errorf("auth: failed to fetch JWKS: %w", err)
	}

	return o, nil
}

// Middleware validates JWT and extracts claims
func (o *OIDC) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := o.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
			return
		}

		// Store claims in context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateToken validates a JWT and returns claims
func (o *OIDC) ValidateToken(tokenString string) (*Claims, error) {
	// Refresh keys if needed (every 1 hour)
	if time.Since(o.lastRefresh) > 1*time.Hour {
		if err := o.refreshKeys(); err != nil {
			// Log error but continue with cached keys
			fmt.Printf("Warning: failed to refresh JWKS: %v\n", err)
		}
	}

	// Parse using MapClaims since our custom Claims doesn't implement jwt.Claims
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		// Look up public key
		o.keysMu.RLock()
		key, exists := o.keys[kid]
		o.keysMu.RUnlock()

		if !exists {
			// Try refreshing keys once
			if err := o.refreshKeys(); err != nil {
				return nil, fmt.Errorf("key %s not found and refresh failed: %w", kid, err)
			}
			o.keysMu.RLock()
			key, exists = o.keys[kid]
			o.keysMu.RUnlock()
			if !exists {
				return nil, fmt.Errorf("key %s not found", kid)
			}
		}

		return key, nil
	})

	if err != nil {
		return nil, err
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Extract issuer and validate
	issuer, _ := mapClaims["iss"].(string)
	if issuer != o.issuer {
		return nil, fmt.Errorf("invalid issuer: %s", issuer)
	}

	// Extract audience and validate (can be string or []string)
	var audiences []string
	switch aud := mapClaims["aud"].(type) {
	case string:
		audiences = []string{aud}
	case []interface{}:
		for _, a := range aud {
			if audStr, ok := a.(string); ok {
				audiences = append(audiences, audStr)
			}
		}
	}

	audienceValid := false
	for _, aud := range audiences {
		if aud == o.audience {
			audienceValid = true
			break
		}
	}
	if !audienceValid {
		return nil, fmt.Errorf("invalid audience: %v", audiences)
	}

	// Convert to our custom Claims struct
	claims := &Claims{
		TenantID: getStringClaim(mapClaims, "tid"),
		Subject:  getStringClaim(mapClaims, "sub"),
		Email:    getStringClaim(mapClaims, "email"),
		Roles:    getStringArrayClaim(mapClaims, "roles"),
		Scopes:   getStringArrayClaim(mapClaims, "scopes"),
		Issuer:   issuer,
		Audience: audiences[0], // Use first audience
		Raw:      mapClaims,
	}

	// Extract expiry
	if exp, ok := mapClaims["exp"].(float64); ok {
		claims.Expiry = time.Unix(int64(exp), 0)
	}

	return claims, nil
}

// Helper to extract string claim
func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

// Helper to extract string array claim
func getStringArrayClaim(claims jwt.MapClaims, key string) []string {
	var result []string

	switch val := claims[key].(type) {
	case []interface{}:
		for _, v := range val {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
	case []string:
		result = val
	case string:
		// Single string, convert to array
		result = []string{val}
	}

	return result
}

func (o *OIDC) refreshKeys() error {
	resp, err := http.Get(o.jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" || key.Use != "sig" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}

		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes).Int64()

		newKeys[key.Kid] = &rsa.PublicKey{
			N: n,
			E: int(e),
		}
	}

	o.keysMu.Lock()
	o.keys = newKeys
	o.lastRefresh = time.Now()
	o.keysMu.Unlock()

	return nil
}

// GetClaims extracts claims from request context
func GetClaims(r *http.Request) (*Claims, error) {
	claims, ok := r.Context().Value("claims").(*Claims)
	if !ok {
		return nil, errors.New("no claims in context")
	}
	return claims, nil
}
