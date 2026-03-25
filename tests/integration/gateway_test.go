package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/viola/tests/integration/testutil"
)

const gatewayBase = "http://localhost:8080"

// TestGateway_Health verifies the gateway-api health endpoint is reachable.
func TestGateway_Health(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(gatewayBase + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /health status=%d body=%s", resp.StatusCode, body)
	}

	t.Log("gateway: /health OK")
}

// TestGateway_UnauthenticatedReturns401 verifies that requests without
// a bearer token are rejected.
func TestGateway_UnauthenticatedReturns401(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(gatewayBase + "/api/v1/alerts")
	if err != nil {
		t.Fatalf("GET /api/v1/alerts: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}

	t.Log("gateway: unauthenticated → 401 OK")
}

// TestGateway_AuthenticatedListAlerts verifies that an authenticated request
// can list alerts via the gateway API.
func TestGateway_AuthenticatedListAlerts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Get a real JWT from the auth service.
	token := getAuthToken(t)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", gatewayBase+"/api/v1/alerts", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/alerts: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	// Verify response is valid JSON.
	var result interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}

	t.Log("gateway: authenticated list alerts OK")
}

// TestGateway_AuthenticatedListIncidents verifies incident listing.
func TestGateway_AuthenticatedListIncidents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := getAuthToken(t)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", gatewayBase+"/api/v1/incidents", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/incidents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	t.Log("gateway: authenticated list incidents OK")
}

// getAuthToken fetches a JWT from the auth service (localhost:8081).
func getAuthToken(t *testing.T) string {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post("http://localhost:8081/token", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /token to auth service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("auth /token status=%d body=%s", resp.StatusCode, body)
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		t.Fatalf("unmarshal token response: %v (body=%s)", err, body)
	}

	token := tokenResp.Token
	if token == "" {
		token = tokenResp.AccessToken
	}
	if token == "" {
		t.Fatalf("no token in auth response: %s", body)
	}

	fmt.Printf("  auth token acquired (len=%d)\n", len(token))
	return token
}
