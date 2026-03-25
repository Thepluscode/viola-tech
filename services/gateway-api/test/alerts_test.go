package test

import (
	"testing"
	"time"

	"github.com/viola/gateway-api/internal/store"
)

func TestAlertEndpoints(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	// Seed test data
	testAlert := &store.Alert{
		TenantID:    h.testTenant,
		AlertID:     "ALT-TEST-001",
		CreatedAt:   time.Now().Add(-30 * time.Minute),
		UpdatedAt:   time.Now(),
		Status:      "open",
		Severity:    "critical",
		Confidence:  0.95,
		RiskScore:   92.0,
		Title:       "Suspicious PowerShell Execution",
		Description: "Detected encoded PowerShell command execution on host WIN-SERVER-01",
		Labels:      map[string]string{"host": "WIN-SERVER-01", "tactic": "execution"},
	}
	alertStore.Upsert(nil, testAlert)

	t.Run("List Alerts - Success", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/alerts", token, nil)
		h.AssertStatus(rr, 200)

		var response map[string]interface{}
		h.AssertJSON(rr, &response)

		alerts, ok := response["alerts"].([]interface{})
		if !ok || len(alerts) == 0 {
			t.Error("Expected at least one alert")
		}
	})

	t.Run("List Alerts - Unauthorized", func(t *testing.T) {
		rr := h.Request("GET", "/api/v1/alerts", "", nil)
		h.AssertStatus(rr, 401)
	})

	t.Run("Get Alert - Success", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/alerts/ALT-TEST-001", token, nil)
		h.AssertStatus(rr, 200)

		var alert store.Alert
		h.AssertJSON(rr, &alert)

		if alert.AlertID != "ALT-TEST-001" {
			t.Errorf("Expected alert ID ALT-TEST-001, got %s", alert.AlertID)
		}
		if alert.Severity != "critical" {
			t.Errorf("Expected severity 'critical', got %s", alert.Severity)
		}
		if alert.Title != "Suspicious PowerShell Execution" {
			t.Errorf("Unexpected alert title: %s", alert.Title)
		}
	})

	t.Run("Get Alert - Not Found", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/alerts/ALT-NONEXISTENT", token, nil)
		h.AssertStatus(rr, 404)
	})

	t.Run("Update Alert - Acknowledge", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		updateBody := map[string]interface{}{
			"status":      "ack",
			"assigned_to": "security-analyst@example.com",
		}

		rr := h.Request("PATCH", "/api/v1/alerts/ALT-TEST-001", token, updateBody)
		h.AssertStatus(rr, 200)

		// Verify update
		alert, _ := alertStore.Get(nil, h.testTenant, "ALT-TEST-001")
		if alert.Status != "ack" {
			t.Errorf("Expected status 'ack', got %s", alert.Status)
		}
	})

	t.Run("Update Alert - Close as False Positive", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		updateBody := map[string]interface{}{
			"status":         "closed",
			"closure_reason": "False positive - authorized admin activity",
		}

		rr := h.Request("PATCH", "/api/v1/alerts/ALT-TEST-001", token, updateBody)
		h.AssertStatus(rr, 200)

		// Verify update
		alert, _ := alertStore.Get(nil, h.testTenant, "ALT-TEST-001")
		if alert.Status != "closed" {
			t.Errorf("Expected status 'closed', got %s", alert.Status)
		}
		if alert.ClosureReason == nil || *alert.ClosureReason != "False positive - authorized admin activity" {
			t.Error("Expected closure_reason to be set correctly")
		}
	})

	t.Run("Update Alert - Forbidden (Reader Cannot Write)", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCReader"},
		})

		updateBody := map[string]interface{}{
			"status": "closed",
		}

		rr := h.Request("PATCH", "/api/v1/alerts/ALT-TEST-001", token, updateBody)
		h.AssertStatus(rr, 403)
	})

	t.Run("Update Alert - Bad Request (Invalid Status)", func(t *testing.T) {
		token := h.CreateTestJWT(map[string]interface{}{
			"roles": []string{"SOCResponder"},
		})

		updateBody := map[string]interface{}{
			"status": "invalid-status-value",
		}

		rr := h.Request("PATCH", "/api/v1/alerts/ALT-TEST-001", token, updateBody)
		h.AssertStatus(rr, 400)
	})
}

func TestAlertWorkflow(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	// Seed test alert
	testAlert := &store.Alert{
		TenantID:    h.testTenant,
		AlertID:     "ALT-WORKFLOW-001",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      "open",
		Severity:    "high",
		Confidence:  0.85,
		RiskScore:   78.0,
		Title:       "Lateral Movement Detected",
		Description: "Suspicious SMB activity between hosts",
		Labels:      map[string]string{},
	}
	alertStore.Upsert(nil, testAlert)

	token := h.CreateTestJWT(map[string]interface{}{
		"roles": []string{"SOCResponder"},
	})

	t.Run("Workflow: Open → Acknowledge → Close", func(t *testing.T) {
		// Step 1: Verify alert is open
		rr := h.Request("GET", "/api/v1/alerts/ALT-WORKFLOW-001", token, nil)
		h.AssertStatus(rr, 200)
		var alert1 store.Alert
		h.AssertJSON(rr, &alert1)
		if alert1.Status != "open" {
			t.Errorf("Expected initial status 'open', got %s", alert1.Status)
		}

		// Step 2: Acknowledge alert
		updateAck := map[string]interface{}{
			"status":      "ack",
			"assigned_to": "analyst@example.com",
		}
		rr = h.Request("PATCH", "/api/v1/alerts/ALT-WORKFLOW-001", token, updateAck)
		h.AssertStatus(rr, 200)

		// Verify acknowledged
		rr = h.Request("GET", "/api/v1/alerts/ALT-WORKFLOW-001", token, nil)
		h.AssertStatus(rr, 200)
		var alert2 store.Alert
		h.AssertJSON(rr, &alert2)
		if alert2.Status != "ack" {
			t.Errorf("Expected status 'ack', got %s", alert2.Status)
		}

		// Step 3: Close alert
		updateClose := map[string]interface{}{
			"status":         "closed",
			"closure_reason": "Confirmed true positive - incident created",
		}
		rr = h.Request("PATCH", "/api/v1/alerts/ALT-WORKFLOW-001", token, updateClose)
		h.AssertStatus(rr, 200)

		// Verify closed
		rr = h.Request("GET", "/api/v1/alerts/ALT-WORKFLOW-001", token, nil)
		h.AssertStatus(rr, 200)
		var alert3 store.Alert
		h.AssertJSON(rr, &alert3)
		if alert3.Status != "closed" {
			t.Errorf("Expected status 'closed', got %s", alert3.Status)
		}
	})
}

func TestAlertMultiTenant(t *testing.T) {
	h := NewTestHelpers(t)
	incidentStore := NewMockIncidentStore()
	alertStore := NewMockAlertStore()
	auditor := NewMockAuditor()

	h.SetupTestRouter(incidentStore, alertStore, auditor)

	// Seed alerts for different tenants
	alert1 := &store.Alert{
		TenantID:  "tenant-1",
		AlertID:   "ALT-TENANT1-001",
		Status:    "open",
		Severity:  "high",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Labels:    map[string]string{},
	}
	alert2 := &store.Alert{
		TenantID:  "tenant-2",
		AlertID:   "ALT-TENANT2-001",
		Status:    "open",
		Severity:  "medium",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Labels:    map[string]string{},
	}
	alertStore.Upsert(nil, alert1)
	alertStore.Upsert(nil, alert2)

	t.Run("Tenant Isolation - Cannot Access Other Tenant's Alerts", func(t *testing.T) {
		// User from tenant-1
		token := h.CreateTestJWT(map[string]interface{}{
			"tid":   "tenant-1",
			"roles": []string{"SOCReader"},
		})

		// Try to access tenant-2's alert
		rr := h.Request("GET", "/api/v1/alerts/ALT-TENANT2-001", token, nil)
		h.AssertStatus(rr, 404) // Should not find it (tenant mismatch)
	})

	t.Run("Tenant Isolation - Can Only List Own Alerts", func(t *testing.T) {
		// User from tenant-1
		token := h.CreateTestJWT(map[string]interface{}{
			"tid":   "tenant-1",
			"roles": []string{"SOCReader"},
		})

		rr := h.Request("GET", "/api/v1/alerts", token, nil)
		h.AssertStatus(rr, 200)

		var response map[string]interface{}
		h.AssertJSON(rr, &response)

		alerts, _ := response["alerts"].([]interface{})
		// Should only see tenant-1's alerts
		// (In real implementation, filtering happens in store layer)
		for _, a := range alerts {
			alertMap := a.(map[string]interface{})
			if tid, ok := alertMap["tenant_id"].(string); ok && tid != "tenant-1" {
				t.Errorf("Found alert from wrong tenant: %s", tid)
			}
		}
	})
}
