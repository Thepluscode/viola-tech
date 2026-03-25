package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/viola/gateway-api/internal/audit"
	apimiddleware "github.com/viola/gateway-api/internal/api/middleware"
	"github.com/viola/gateway-api/internal/store"
	auditv1 "github.com/viola/shared/proto/audit"
)

type AlertHandlers struct {
	store   *store.AlertStore
	auditor *audit.Emitter
}

func NewAlertHandlers(s *store.AlertStore, auditor *audit.Emitter) *AlertHandlers {
	return &AlertHandlers{store: s, auditor: auditor}
}

func (h *AlertHandlers) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusUnauthorized, "Missing tenant ID", "UNAUTHORIZED")
		return
	}

	filter := store.ListAlertsFilter{
		Status:   r.URL.Query().Get("status"),
		Severity: r.URL.Query().Get("severity"),
		Limit:    50,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	// Cursor-based pagination: ?cursor=<rfc3339>|<alert_id>
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				filter.AfterUpdatedAt = t
				filter.AfterID = parts[1]
			}
		}
	} else if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	result, err := h.store.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list alerts", "INTERNAL_ERROR")
		return
	}

	resp := map[string]interface{}{
		"alerts": result.Alerts,
		"count":  len(result.Alerts),
	}
	if result.NextCursor != "" {
		resp["next_cursor"] = result.NextCursor
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AlertHandlers) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusUnauthorized, "Missing tenant ID", "UNAUTHORIZED")
		return
	}

	alertID := chi.URLParam(r, "id")
	if alertID == "" {
		writeError(w, http.StatusBadRequest, "Missing alert ID", "BAD_REQUEST")
		return
	}

	alert, err := h.store.Get(r.Context(), tenantID, alertID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get alert", "INTERNAL_ERROR")
		return
	}
	if alert == nil {
		writeError(w, http.StatusNotFound, "Alert not found", "NOT_FOUND")
		return
	}

	// C3: defence-in-depth tenant isolation check.
	if !requireTenantMatch(w, tenantID, alert.TenantID) {
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

type UpdateAlertRequest struct {
	Status        *string `json:"status,omitempty"`
	AssignedTo    *string `json:"assigned_to,omitempty"`
	ClosureReason *string `json:"closure_reason,omitempty"`
}

func (h *AlertHandlers) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusUnauthorized, "Missing tenant ID", "UNAUTHORIZED")
		return
	}

	alertID := chi.URLParam(r, "id")
	if alertID == "" {
		writeError(w, http.StatusBadRequest, "Missing alert ID", "BAD_REQUEST")
		return
	}

	var req UpdateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", "BAD_REQUEST")
		return
	}

	updates := make(map[string]interface{})
	action := "update"
	if req.Status != nil {
		if *req.Status != "open" && *req.Status != "ack" && *req.Status != "closed" {
			writeError(w, http.StatusBadRequest, "Invalid status (must be open|ack|closed)", "BAD_REQUEST")
			return
		}
		updates["status"] = *req.Status
		// Specific actions for audit
		if *req.Status == "ack" {
			action = "acknowledge"
		} else if *req.Status == "closed" {
			action = "close"
		}
	}
	if req.AssignedTo != nil {
		updates["assigned_to"] = *req.AssignedTo
		action = "assign"
	}
	if req.ClosureReason != nil {
		updates["closure_reason"] = *req.ClosureReason
	}

	if len(updates) == 0 {
		writeError(w, http.StatusBadRequest, "No updates provided", "BAD_REQUEST")
		return
	}

	if err := h.store.Update(r.Context(), tenantID, alertID, updates); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update alert", "INTERNAL_ERROR")

		// Emit audit event for failure
		h.emitAudit(r, tenantID, "alert", alertID, action, "failure", updates)
		return
	}

	// Emit audit event for success
	h.emitAudit(r, tenantID, "alert", alertID, action, "success", updates)

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AlertHandlers) emitAudit(r *http.Request, tenantID, resourceType, resourceID, action, outcome string, metadata map[string]interface{}) {
	if h.auditor == nil {
		return // Auditing not configured
	}

	// Extract actor info from claims
	actorID := "unknown"
	actorType := "user"
	if claims, ok := apimiddleware.ClaimsFrom(r.Context()); ok {
		actorID = claims.Email
		if actorID == "" {
			actorID = claims.Subject
		}
	}

	// Get request ID
	requestID := middleware.GetReqID(r.Context())

	// Convert metadata to string map
	metaStr := make(map[string]string)
	for k, v := range metadata {
		metaStr[k] = toJSONString(v)
	}

	ev := &auditv1.AuditEvent{
		ActorType:    actorType,
		ActorId:      actorID,
		ActorIp:      r.RemoteAddr,
		ResourceType: resourceType,
		ResourceId:   resourceID,
		Action:       action,
		Outcome:      outcome,
		Metadata:     metaStr,
	}

	_ = h.auditor.Emit(r.Context(), tenantID, requestID, ev)
}

// toJSONString is defined in common.go (C2: single canonical helper).
