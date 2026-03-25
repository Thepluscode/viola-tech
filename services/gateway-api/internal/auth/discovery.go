package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OIDCDiscovery represents OpenID Connect discovery document
type OIDCDiscovery struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

// DiscoverJWKSURL fetches the JWKS URL from the OIDC issuer's discovery endpoint
func DiscoverJWKSURL(ctx context.Context, issuerURL string) (string, error) {
	// Ensure issuer URL doesn't end with /
	issuerURL = strings.TrimSuffix(issuerURL, "/")

	// OIDC discovery endpoint
	discoveryURL := issuerURL + "/.well-known/openid-configuration"

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch OIDC discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery returned status %d", resp.StatusCode)
	}

	var doc OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("failed to decode OIDC discovery: %w", err)
	}

	if doc.JWKSURI == "" {
		return "", fmt.Errorf("OIDC discovery did not contain jwks_uri")
	}

	return doc.JWKSURI, nil
}
