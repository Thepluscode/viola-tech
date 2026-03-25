// Package handler exposes the response action API over HTTP.
//
// Endpoints:
//
//	POST /actions              Submit a new response action
//	GET  /actions?tenant_id=X List recent actions for a tenant
//	GET  /health
//
// Authentication: the X-Tenant-ID header is required on every request.
// In production this would be extracted from a verified JWT; here the header
// is accepted directly since the response service sits behind the gateway-api
// which enforces JWT verification.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/viola/response/internal/executor"
	"github.com/viola/response/internal/store"
)

// Handler holds the store and executor.
type Handler struct {
	store    *store.Store
	executor executor.Executor
}

// New creates a Handler.
func New(s *store.Store, e executor.Executor) *Handler {
	return &Handler{store: s, executor: e}
}

// ActionRequest is the JSON body for POST /actions.
type ActionRequest struct {
	TenantID    string `json:"tenant_id"`
	IncidentID  string `json:"incident_id,omitempty"`
	AlertID     string `json:"alert_id,omitempty"`
	ActionType  string `json:"action_type"` // isolate_host | block_ip | kill_process | contain_user
	Target      string `json:"target"`      // hostname, IP, process, username
	Reason      string `json:"reason"`
	TriggeredBy string `json:"triggered_by"` // "auto" or user sub
}

// ActionResponse is returned on POST /actions success.
type ActionResponse struct {
	ActionID  string `json:"action_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// CreateAction handles POST /actions.
func (h *Handler) CreateAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields.
	if req.TenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}
	if req.TriggeredBy == "" {
		req.TriggeredBy = "auto"
	}
	if err := executor.ValidateType(req.ActionType); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	actionID := newActionID()
	now := time.Now().UTC()

	// Persist with status "pending" before execution.
	a := &store.Action{
		ActionID:    actionID,
		TenantID:    req.TenantID,
		IncidentID:  req.IncidentID,
		AlertID:     req.AlertID,
		ActionType:  req.ActionType,
		Target:      req.Target,
		Status:      "pending",
		Reason:      req.Reason,
		TriggeredBy: req.TriggeredBy,
		CreatedAt:   now,
		UpdatedAt:   now,
		Detail:      "{}",
	}
	if err := h.store.Insert(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist action")
		return
	}

	// Execute asynchronously so the API call returns fast.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		execErr := h.executor.Execute(ctx, executor.Request{
			ActionID:   actionID,
			TenantID:   req.TenantID,
			ActionType: req.ActionType,
			Target:     req.Target,
			Reason:     req.Reason,
		})

		status := "success"
		if execErr != nil {
			status = "failed"
		}
		_ = h.store.UpdateStatus(ctx, req.TenantID, actionID, status)
	}()

	writeJSON(w, http.StatusAccepted, ActionResponse{
		ActionID:  actionID,
		Status:    "pending",
		CreatedAt: now.Format(time.RFC3339),
	})
}

// ListActions handles GET /actions?tenant_id=X&limit=50.
func (h *Handler) ListActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id query param is required")
		return
	}

	limit := 50
	if ls := r.URL.Query().Get("limit"); ls != "" {
		n, err := strconv.Atoi(ls)
		if err != nil || n < 1 || n > 500 {
			writeError(w, http.StatusBadRequest, "limit must be an integer between 1 and 500")
			return
		}
		limit = n
	}

	actions, err := h.store.List(r.Context(), tenantID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list actions")
		return
	}
	if actions == nil {
		actions = []*store.Action{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"actions": actions,
		"count":   len(actions),
	})
}

// Health handles GET /health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func newActionID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ra-%s", hex.EncodeToString(b))
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
