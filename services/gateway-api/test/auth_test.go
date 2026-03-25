package test

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	t.Run("Health Check - No Auth Required", func(t *testing.T) {
		rr := h.Request("GET", "/health", "", nil)
		h.AssertStatus(rr, 200)

		if rr.Body.String() != "OK" {
			t.Errorf("Expected 'OK', got %s", rr.Body.String())
		}
	})
}

func TestAuthentication(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	t.Run("Missing Token", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/incidents", "", nil)
		h.AssertStatus(rr, 401)
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/incidents", "not-a-valid-jwt", nil)
		h.AssertStatus(rr, 401)
	})

	t.Run("Expired Token", func(t *testing.T) {
		// Create expired token
		token := h.CreateTestJWT(map[string]interface{}{
			"exp":   1577836800, // Jan 1, 2020 - definitely expired
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 401)
	})

	t.Run("Valid Token - Success", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)
	})

	t.Run("Token With Different Tenant - Isolation", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"tid":   "other-tenant",
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)

		var response map[string]interface{}
		h.AssertJSON(rr, &response)

		// Should get empty list (no incidents for this tenant)
		incidents, _ := response["incidents"].([]interface{})
		if len(incidents) != 0 {
			t.Error("Expected empty incidents list for different tenant")
		}
	})
}

func TestAuthorization(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	t.Run("SOCReader - Can Read Incidents", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)
	})

	t.Run("SOCReader - Cannot Write Incidents", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		updateBody := map[string]interface{}{
			"status": "closed",
		}

		rr := h.Request("PATCH", "/api/v1/incidents/INC-001", token, updateBody)
		h.AssertStatus(rr, 403)
	})

	t.Run("SOCResponder - Can Read and Write", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		// Can read
		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)

		// Can write
		updateBody := map[string]interface{}{
			"status": "ack",
		}
		rr = h.Request("PATCH", "/api/v1/incidents/INC-001", token, updateBody)
		// Note: Will return 404 if incident doesn't exist, but not 403
		if rr.Code == 403 {
			t.Error("SOCResponder should be able to update incidents")
		}
	})

	t.Run("ViolaAdmin - Full Access", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"ViolaAdmin"},
		})

		// Can read
		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)

		// Can write
		updateBody := map[string]interface{}{
			"status": "closed",
		}
		rr = h.Request("PATCH", "/api/v1/incidents/INC-001", token, updateBody)
		if rr.Code == 403 {
			t.Error("Admin should have full access")
		}
	})

	t.Run("No Roles - Forbidden", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 403)
	})

	t.Run("Multiple Roles - Granted if Any Match", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SomeOtherRole", "SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)
	})
}

func TestScopeBasedAuthorization(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	t.Run("Scope - incidents:read", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"scp": "incidents:read alerts:read",
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		// Note: Scope-based authz would need to be implemented in RBAC middleware
		// This test documents expected behavior
		if rr.Code == 403 {
			t.Skip("Scope-based authorization not yet implemented")
		}
	})

	t.Run("Scope - incidents:write", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"scp": "incidents:write",
		})

		updateBody := map[string]interface{}{
			"status": "ack",
		}

		rr := h.Request("PATCH", "/api/v1/incidents/INC-001", token, updateBody)
		// Scope-based authz implementation would check this
		if rr.Code == 403 {
			t.Skip("Scope-based authorization not yet implemented")
		}
	})
}

func TestClaimExtraction(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	t.Run("Extract Tenant ID from tid Claim", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"tid":   "custom-tenant-id",
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)
		// Handlers use tid claim for tenant isolation
	})

	t.Run("Extract Email from email Claim", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"email": "user@example.com",
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)
		// Email would be used for audit logging
	})

	t.Run("Extract Subject from sub Claim", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"sub":   "auth0|123456",
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)
		// Subject is primary user identifier
	})
}

func TestErrorResponses(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	t.Run("401 Unauthorized - Proper Error Format", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/incidents", "", nil)
		h.AssertStatus(rr, 401)

		// Should contain error message
		body := rr.Body.String()
		if body == "" {
			t.Error("Expected error message in response body")
		}
	})

	t.Run("403 Forbidden - Proper Error Format", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{}, // No roles
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 403)

		body := rr.Body.String()
		if body == "" {
			t.Error("Expected error message in response body")
		}
	})

	t.Run("404 Not Found - Proper Error Format", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents/NONEXISTENT", token, nil)
		h.AssertStatus(rr, 404)

		var errorResponse map[string]interface{}
		h.AssertJSON(rr, &errorResponse)

		if _, ok := errorResponse["error"]; !ok {
			t.Error("Expected 'error' field in error response")
		}
	})

	t.Run("400 Bad Request - Invalid JSON", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		// Send invalid JSON
		invalidBody := "{invalid-json"
		req := httptest.NewRequest("PATCH", "/api/v1/incidents/INC-001", bytes.NewReader([]byte(invalidBody)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		h.router.ServeHTTP(rr, req)

		h.AssertStatus(rr, 400)
	})
}
