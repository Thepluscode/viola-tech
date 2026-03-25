package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IntrospectionClient checks token validity with the IdP
type IntrospectionClient struct {
	endpoint     string
	clientID     string
	clientSecret string
	http         *http.Client
}

// IntrospectionConfig configures token introspection
type IntrospectionConfig struct {
	Endpoint     string // e.g., "https://login.microsoftonline.com/<tenant>/oauth2/v2.0/introspect"
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

// IntrospectionResponse represents the introspection response
type IntrospectionResponse struct {
	Active bool   `json:"active"`
	Scope  string `json:"scope,omitempty"`
	Sub    string `json:"sub,omitempty"`
	Exp    int64  `json:"exp,omitempty"`
	Iat    int64  `json:"iat,omitempty"`

	// Additional fields (IdP-specific)
	ClientID string `json:"client_id,omitempty"`
	Username string `json:"username,omitempty"`
	TokenType string `json:"token_type,omitempty"`
}

// NewIntrospectionClient creates a new introspection client
func NewIntrospectionClient(cfg IntrospectionConfig) (*IntrospectionClient, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("introspection endpoint required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("client ID required")
	}
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("client secret required")
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &IntrospectionClient{
		endpoint:     cfg.Endpoint,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		http:         cfg.HTTPClient,
	}, nil
}

// Introspect checks if a token is still valid (not revoked)
func (c *IntrospectionClient) Introspect(ctx context.Context, token string) (*IntrospectionResponse, error) {
	// Build form data
	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspection returned status %d", resp.StatusCode)
	}

	var result IntrospectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode introspection response: %w", err)
	}

	return &result, nil
}

// IsActive checks if a token is active (not revoked or expired)
func (c *IntrospectionClient) IsActive(ctx context.Context, token string) (bool, error) {
	result, err := c.Introspect(ctx, token)
	if err != nil {
		return false, err
	}
	return result.Active, nil
}
