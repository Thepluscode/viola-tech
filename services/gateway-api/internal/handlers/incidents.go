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

type IncidentHandlers struct {
	store   *store.IncidentStore
	auditor *audit.Emitter
}

func NewIncidentHandlers(s *store.IncidentStore, auditor *audit.Emitter) *IncidentHandlers {
	return &IncidentHandlers{store: s, auditor: auditor}
}

func (h *IncidentHandlers) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusUnauthorized, "Missing tenant ID", "UNAUTHORIZED")
		return
	}

	filter := store.ListIncidentsFilter{
		Status:   r.URL.Query().Get("status"),
		Severity: r.URL.Query().Get("severity"),
		Limit:    50,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	// Cursor-based pagination: ?cursor=<rfc3339>|<incident_id>
	// Falls back to legacy offset when cursor is absent.
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
		writeError(w, http.StatusInternalServerError, "Failed to list incidents", "INTERNAL_ERROR")
		return
	}

	resp := map[string]interface{}{
		"incidents": result.Incidents,
		"count":     len(result.Incidents),
	}
	if result.NextCursor != "" {
		resp["next_cursor"] = result.NextCursor
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *IncidentHandlers) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusUnauthorized, "Missing tenant ID", "UNAUTHORIZED")
		return
	}

	incidentID := chi.URLParam(r, "id")
	if incidentID == "" {
		writeError(w, http.StatusBadRequest, "Missing incident ID", "BAD_REQUEST")
		return
	}

	incident, err := h.store.Get(r.Context(), tenantID, incidentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get incident", "INTERNAL_ERROR")
		return
	}
	if incident == nil {
		writeError(w, http.StatusNotFound, "Incident not found", "NOT_FOUND")
		return
	}

	// C3: defence-in-depth tenant isolation check.
	if !requireTenantMatch(w, tenantID, incident.TenantID) {
		return
	}

	writeJSON(w, http.StatusOK, incident)
}

type UpdateIncidentRequest struct {
	Status        *string `json:"status,omitempty"`
	AssignedTo    *string `json:"assigned_to,omitempty"`
	ClosureReason *string `json:"closure_reason,omitempty"`
}

func (h *IncidentHandlers) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		writeError(w, http.StatusUnauthorized, "Missing tenant ID", "UNAUTHORIZED")
		return
	}

	incidentID := chi.URLParam(r, "id")
	if incidentID == "" {
		writeError(w, http.StatusBadRequest, "Missing incident ID", "BAD_REQUEST")
		return
	}

	var req UpdateIncidentRequest
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

	if err := h.store.Update(r.Context(), tenantID, incidentID, updates); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update incident", "INTERNAL_ERROR")

		// Emit audit event for failure
		h.emitAudit(r, tenantID, "incident", incidentID, action, "failure", updates)
		return
	}

	// Emit audit event for success
	h.emitAudit(r, tenantID, "incident", incidentID, action, "success", updates)

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *IncidentHandlers) emitAudit(r *http.Request, tenantID, resourceType, resourceID, action, outcome string, metadata map[string]interface{}) {
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
