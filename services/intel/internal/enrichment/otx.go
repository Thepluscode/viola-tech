// Package enrichment queries the AlienVault OTX threat intelligence API.
//
// Supported indicator types:
//
//	ip   — IPv4 address lookup (general reputation + pulse count)
//	domain — domain lookup
//	hash — file hash lookup (MD5, SHA1, SHA256 auto-detected)
//
// All lookups are fire-and-forget enrichment: results are returned as a flat
// Result struct. On any API error the Reputation field stays 0 and the error
// is surfaced to the caller for logging.
package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	otxBaseURL = "https://otx.alienvault.com/api/v1"
	// defaultTimeout guards individual OTX API calls.
	defaultTimeout = 8 * time.Second
)

// Result is the normalised enrichment response returned to callers.
type Result struct {
	Indicator  string   `json:"indicator"`
	Type       string   `json:"type"`   // "ip" | "domain" | "hash"
	Reputation int      `json:"reputation"` // pulse count; 0 = unknown / benign
	Malicious  bool     `json:"malicious"`  // true if pulse_count > 0
	Country    string   `json:"country,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Source     string   `json:"source"` // always "otx"
}

// Client wraps the OTX HTTP client.
type Client struct {
	http    *http.Client
	apiKey  string // optional — OTX allows unauthenticated basic lookups
	baseURL string
}

// NewClient creates an OTX client.  apiKey may be empty for unauthenticated
// lookups (rate-limited but sufficient for dev/demo).
func NewClient(apiKey string) *Client {
	return &Client{
		http:    &http.Client{Timeout: defaultTimeout},
		apiKey:  apiKey,
		baseURL: otxBaseURL,
	}
}

// LookupIP fetches reputation data for an IPv4 address.
func (c *Client) LookupIP(ctx context.Context, ip string) (*Result, error) {
	return c.lookup(ctx, "ip", ip, fmt.Sprintf("%s/indicators/IPv4/%s/general", c.baseURL, ip))
}

// LookupDomain fetches reputation data for a domain name.
func (c *Client) LookupDomain(ctx context.Context, domain string) (*Result, error) {
	return c.lookup(ctx, "domain", domain, fmt.Sprintf("%s/indicators/domain/%s/general", c.baseURL, domain))
}

// LookupHash fetches reputation data for a file hash (MD5/SHA1/SHA256).
func (c *Client) LookupHash(ctx context.Context, hash string) (*Result, error) {
	// OTX uses the same "file" section for all hash types.
	return c.lookup(ctx, "hash", hash, fmt.Sprintf("%s/indicators/file/%s/general", c.baseURL, hash))
}

// otxGeneralResponse is the subset of fields we use from OTX /general.
type otxGeneralResponse struct {
	PulseInfo struct {
		Count int `json:"count"`
		Pulses []struct {
			Tags []string `json:"tags"`
		} `json:"pulses"`
	} `json:"pulse_info"`
	CountryCode string `json:"country_code"`
}

func (c *Client) lookup(ctx context.Context, iocType, indicator, url string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("otx: build request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-OTX-API-KEY", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("otx: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("otx: rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("otx: unexpected status %d for %s", resp.StatusCode, indicator)
	}

	var body otxGeneralResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("otx: decode: %w", err)
	}

	// Collect unique tags from first 10 pulses (avoid huge tag sets).
	tagSet := make(map[string]struct{})
	limit := len(body.PulseInfo.Pulses)
	if limit > 10 {
		limit = 10
	}
	for _, p := range body.PulseInfo.Pulses[:limit] {
		for _, t := range p.Tags {
			tagSet[strings.ToLower(t)] = struct{}{}
		}
	}
	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}

	pulseCount := body.PulseInfo.Count
	return &Result{
		Indicator:  indicator,
		Type:       iocType,
		Reputation: pulseCount,
		Malicious:  pulseCount > 0,
		Country:    body.CountryCode,
		Tags:       tags,
		Source:     "otx",
	}, nil
}
