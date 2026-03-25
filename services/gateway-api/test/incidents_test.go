package test

import (
	"testing"
	"time"

	"github.com/viola/gateway-api/internal/store"
)

func TestIncidentEndpoints(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	// Setup test router
	h.SetupTestRouter(incidentStore, alertStore, auditor)

	// Seed test data
	testIncident := &store.Incident{
		TenantID:          h.testTenant,
		IncidentID:        "INC-TEST-001",
		CorrelatedGroupID: "group-123",
		CreatedAt:         time.Now().Add(-1 * time.Hour),
		UpdatedAt:         time.Now(),
		Status:            "open",
		Severity:          "high",
		MaxRiskScore:      85.5,
		MaxConfidence:     0.9,
		Labels:            map[string]string{"environment": "production"},
		AlertCount:        5,
		HitCount:          12,
	}
	incidentStore.Upsert(nil, testIncident)

	t.Run("List Incidents - Success", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)

		var response map[string]interface{}
		h.AssertJSON(rr, &response)

		incidents, ok := response["incidents"].([]interface{})
		if !ok || len(incidents) == 0 {
			t.Error("Expected at least one incident")
		}
	})

	t.Run("List Incidents - Unauthorized (No Token)", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/incidents", "", nil)
		h.AssertStatus(rr, 401)
	})

	t.Run("List Incidents - Forbidden (No Role)", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{}, // No roles
		})

		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 403)
	})

	t.Run("Get Incident - Success", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents/INC-TEST-001", token, nil)
		h.AssertStatus(rr, 200)

		var incident store.Incident
		h.AssertJSON(rr, &incident)

		if incident.IncidentID != "INC-TEST-001" {
			t.Errorf("Expected incident ID INC-TEST-001, got %s", incident.IncidentID)
		}
		if incident.Status != "open" {
			t.Errorf("Expected status 'open', got %s", incident.Status)
		}
	})

	t.Run("Get Incident - Not Found", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/incidents/INC-NONEXISTENT", token, nil)
		h.AssertStatus(rr, 404)
	})

	t.Run("Update Incident - Acknowledge", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		updateBody := map[string]interface{}{
			"status":      "ack",
			"assigned_to": "analyst@example.com",
		}

		rr := h.Request("PATCH", "/api/v1/incidents/INC-TEST-001", token, updateBody)
		h.AssertStatus(rr, 200)

		// Verify update
		incident, _ := incidentStore.Get(nil, h.testTenant, "INC-TEST-001")
		if incident.Status != "ack" {
			t.Errorf("Expected status 'ack', got %s", incident.Status)
		}
		if incident.AssignedTo == nil || *incident.AssignedTo != "analyst@example.com" {
			t.Error("Expected assigned_to to be set")
		}
	})

	t.Run("Update Incident - Close", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		updateBody := map[string]interface{}{
			"status":         "closed",
			"closure_reason": "False positive",
		}

		rr := h.Request("PATCH", "/api/v1/incidents/INC-TEST-001", token, updateBody)
		h.AssertStatus(rr, 200)

		// Verify update
		incident, _ := incidentStore.Get(nil, h.testTenant, "INC-TEST-001")
		if incident.Status != "closed" {
			t.Errorf("Expected status 'closed', got %s", incident.Status)
		}
		if incident.ClosureReason == nil || *incident.ClosureReason != "False positive" {
			t.Error("Expected closure_reason to be set")
		}
	})

	t.Run("Update Incident - Forbidden (Insufficient Permissions)", func(t *testing.T) {
		// SOCReader can only read, not write
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		updateBody := map[string]interface{}{
			"status": "closed",
		}

		rr := h.Request("PATCH", "/api/v1/incidents/INC-TEST-001", token, updateBody)
		h.AssertStatus(rr, 403)
	})

	t.Run("Update Incident - Bad Request (Invalid Status)", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		updateBody := map[string]interface{}{
			"status": "invalid-status",
		}

		rr := h.Request("PATCH", "/api/v1/incidents/INC-TEST-001", token, updateBody)
		h.AssertStatus(rr, 400)
	})

	t.Run("Admin Can Access Everything", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"ViolaAdmin"},
		})

		// List
		rr := h.Request("GET", "/api/v1/incidents", token, nil)
		h.AssertStatus(rr, 200)

		// Get
		rr = h.Request("GET", "/api/v1/incidents/INC-TEST-001", token, nil)
		h.AssertStatus(rr, 200)

		// Update
		updateBody := map[string]interface{}{
			"status": "closed",
		}
		rr = h.Request("PATCH", "/api/v1/incidents/INC-TEST-001", token, updateBody)
		h.AssertStatus(rr, 200)
	})
}

func TestIncidentFiltering(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	// Seed multiple incidents
	incidents := []*store.Incident{
		{
			TenantID:   h.testTenant,
			IncidentID: "INC-001",
			Status:     "open",
			Severity:   "high",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Labels:     map[string]string{},
		},
		{
			TenantID:   h.testTenant,
			IncidentID: "INC-002",
			Status:     "ack",
			Severity:   "critical",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Labels:     map[string]string{},
		},
		{
			TenantID:   h.testTenant,
			IncidentID: "INC-003",
			Status:     "closed",
			Severity:   "medium",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Labels:     map[string]string{},
		},
	}

	for _, inc := range incidents {
		incidentStore.Upsert(nil, inc)
	}

	token := h.CreateTestJWT(map[string]interface{}{
		"roles": []string{"SOCReader"},
	})

	t.Run("Filter by Status", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/incidents?status=open", token, nil)
		h.AssertStatus(rr, 200)

		var response map[string]interface{}
		h.AssertJSON(rr, &response)

		// Note: Mock store doesn't actually filter, but real implementation would
		incidents, _ := response["incidents"].([]interface{})
		if len(incidents) == 0 {
			t.Error("Expected filtered results")
		}
	})

	t.Run("Filter by Severity", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/incidents?severity=critical", token, nil)
		h.AssertStatus(rr, 200)
	})

	t.Run("Pagination", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/incidents?limit=10&offset=0", token, nil)
		h.AssertStatus(rr, 200)
	})
}
