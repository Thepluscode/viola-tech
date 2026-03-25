package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Incident struct {
	TenantID           string            `json:"tenant_id"`
	IncidentID         string            `json:"incident_id"`
	CorrelatedGroupID  string            `json:"correlated_group_id"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Status             string            `json:"status"`
	Severity           string            `json:"severity"`
	MaxRiskScore       float64           `json:"max_risk_score"`
	MaxConfidence      float64           `json:"max_confidence"`
	MitreTactic        *string           `json:"mitre_tactic,omitempty"`
	MitreTechnique     *string           `json:"mitre_technique,omitempty"`
	Labels             map[string]string `json:"labels"`
	AssignedTo         *string           `json:"assigned_to,omitempty"`
	ClosureReason      *string           `json:"closure_reason,omitempty"`
	RequestID          *string           `json:"request_id,omitempty"`
	AlertCount         int               `json:"alert_count"`
	HitCount           int               `json:"hit_count"`
	EntityIDs          []string          `json:"entity_ids"`
	AlertIDs           []string          `json:"alert_ids"`
	DetectionHitIDs    []string          `json:"detection_hit_ids"`
}

type IncidentStore struct {
	pool *pgxpool.Pool
}

func NewIncidentStore(pool *pgxpool.Pool) *IncidentStore {
	return &IncidentStore{pool: pool}
}

func (s *IncidentStore) Upsert(ctx context.Context, inc *Incident) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	labelsJSON, err := json.Marshal(inc.Labels)
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	// Upsert main incident record
	q := `
		INSERT INTO incidents (
			tenant_id, incident_id, correlated_group_id, created_at, updated_at,
			status, severity, max_risk_score, max_confidence,
			mitre_tactic, mitre_technique, labels, assigned_to, closure_reason, request_id,
			alert_count, hit_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (tenant_id, incident_id) DO UPDATE SET
			updated_at = EXCLUDED.updated_at,
			status = EXCLUDED.status,
			severity = EXCLUDED.severity,
			max_risk_score = EXCLUDED.max_risk_score,
			max_confidence = EXCLUDED.max_confidence,
			mitre_tactic = EXCLUDED.mitre_tactic,
			mitre_technique = EXCLUDED.mitre_technique,
			labels = EXCLUDED.labels,
			assigned_to = EXCLUDED.assigned_to,
			closure_reason = EXCLUDED.closure_reason,
			alert_count = EXCLUDED.alert_count,
			hit_count = EXCLUDED.hit_count
	`
	_, err = tx.Exec(ctx, q,
		inc.TenantID, inc.IncidentID, inc.CorrelatedGroupID, inc.CreatedAt, inc.UpdatedAt,
		inc.Status, inc.Severity, inc.MaxRiskScore, inc.MaxConfidence,
		inc.MitreTactic, inc.MitreTechnique, labelsJSON, inc.AssignedTo, inc.ClosureReason, inc.RequestID,
		inc.AlertCount, inc.HitCount,
	)
	if err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}

	// Upsert entity associations
	if len(inc.EntityIDs) > 0 {
		for _, entityID := range inc.EntityIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO incident_entities (tenant_id, incident_id, entity_id)
				VALUES ($1, $2, $3)
				ON CONFLICT (tenant_id, incident_id, entity_id) DO NOTHING
			`, inc.TenantID, inc.IncidentID, entityID)
			if err != nil {
				return fmt.Errorf("upsert entity: %w", err)
			}
		}
	}

	// Upsert alert associations
	if len(inc.AlertIDs) > 0 {
		for _, alertID := range inc.AlertIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO incident_alerts (tenant_id, incident_id, alert_id)
				VALUES ($1, $2, $3)
				ON CONFLICT (tenant_id, incident_id, alert_id) DO NOTHING
			`, inc.TenantID, inc.IncidentID, alertID)
			if err != nil {
				return fmt.Errorf("upsert alert: %w", err)
			}
		}
	}

	// Upsert hit associations
	if len(inc.DetectionHitIDs) > 0 {
		for _, hitID := range inc.DetectionHitIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO incident_hits (tenant_id, incident_id, hit_id)
				VALUES ($1, $2, $3)
				ON CONFLICT (tenant_id, incident_id, hit_id) DO NOTHING
			`, inc.TenantID, inc.IncidentID, hitID)
			if err != nil {
				return fmt.Errorf("upsert hit: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *IncidentStore) Get(ctx context.Context, tenantID, incidentID string) (*Incident, error) {
	q := `
		SELECT tenant_id, incident_id, correlated_group_id, created_at, updated_at,
			status, severity, max_risk_score, max_confidence,
			mitre_tactic, mitre_technique, labels, assigned_to, closure_reason, request_id,
			alert_count, hit_count
		FROM incidents
		WHERE tenant_id = $1 AND incident_id = $2
	`
	var inc Incident
	var labelsJSON []byte
	err := s.pool.QueryRow(ctx, q, tenantID, incidentID).Scan(
		&inc.TenantID, &inc.IncidentID, &inc.CorrelatedGroupID, &inc.CreatedAt, &inc.UpdatedAt,
		&inc.Status, &inc.Severity, &inc.MaxRiskScore, &inc.MaxConfidence,
		&inc.MitreTactic, &inc.MitreTechnique, &labelsJSON, &inc.AssignedTo, &inc.ClosureReason, &inc.RequestID,
		&inc.AlertCount, &inc.HitCount,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query incident: %w", err)
	}

	if err := json.Unmarshal(labelsJSON, &inc.Labels); err != nil {
		return nil, fmt.Errorf("unmarshal labels: %w", err)
	}

	// Load entities
	rows, err := s.pool.Query(ctx, `SELECT entity_id FROM incident_entities WHERE tenant_id = $1 AND incident_id = $2`, tenantID, incidentID)
	if err != nil {
		return nil, fmt.Errorf("query entities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var eid string
		if err := rows.Scan(&eid); err != nil {
			return nil, err
		}
		inc.EntityIDs = append(inc.EntityIDs, eid)
	}

	// Load alerts
	rows, err = s.pool.Query(ctx, `SELECT alert_id FROM incident_alerts WHERE tenant_id = $1 AND incident_id = $2`, tenantID, incidentID)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var aid string
		if err := rows.Scan(&aid); err != nil {
			return nil, err
		}
		inc.AlertIDs = append(inc.AlertIDs, aid)
	}

	// Load hits
	rows, err = s.pool.Query(ctx, `SELECT hit_id FROM incident_hits WHERE tenant_id = $1 AND incident_id = $2`, tenantID, incidentID)
	if err != nil {
		return nil, fmt.Errorf("query hits: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var hid string
		if err := rows.Scan(&hid); err != nil {
			return nil, err
		}
		inc.DetectionHitIDs = append(inc.DetectionHitIDs, hid)
	}

	return &inc, nil
}

// ListIncidentsFilter controls which incidents are returned.
// Cursor-based pagination is preferred over offset:
//   - Set AfterUpdatedAt + AfterID to the last row's (updated_at, incident_id)
//     from the previous page to fetch the next page in O(log N) index seeks.
//   - Fall back to Offset when cursor fields are zero (backwards-compatible).
type ListIncidentsFilter struct {
	Status   string
	Severity string
	Limit    int

	// Cursor pagination (keyset): supply both fields from the previous page's
	// last row. Zero value disables cursor and falls back to Offset.
	AfterUpdatedAt time.Time
	AfterID        string

	// Legacy offset pagination (avoid for large tables).
	Offset int
}

// ListResult wraps the page rows and an opaque cursor string for the next page.
type IncidentListResult struct {
	Incidents  []*Incident
	NextCursor string // "<rfc3339>|<incident_id>" — empty when no next page
}

func (s *IncidentStore) List(ctx context.Context, tenantID string, filter ListIncidentsFilter) (*IncidentListResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	q := `
		SELECT tenant_id, incident_id, correlated_group_id, created_at, updated_at,
			status, severity, max_risk_score, max_confidence,
			mitre_tactic, mitre_technique, labels, assigned_to, closure_reason, request_id,
			alert_count, hit_count
		FROM incidents
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argPos := 2

	if filter.Status != "" {
		q += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}
	if filter.Severity != "" {
		q += fmt.Sprintf(" AND severity = $%d", argPos)
		args = append(args, filter.Severity)
		argPos++
	}

	if !filter.AfterUpdatedAt.IsZero() && filter.AfterID != "" {
		// Keyset pagination: skip rows at or after the cursor position.
		// Uses the composite index on (tenant_id, updated_at DESC, incident_id).
		q += fmt.Sprintf(
			" AND (updated_at, incident_id) < ($%d, $%d)",
			argPos, argPos+1,
		)
		args = append(args, filter.AfterUpdatedAt, filter.AfterID)
		argPos += 2
		q += fmt.Sprintf(" ORDER BY updated_at DESC, incident_id DESC LIMIT $%d", argPos)
		args = append(args, filter.Limit)
	} else {
		q += fmt.Sprintf(" ORDER BY updated_at DESC, incident_id DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
		args = append(args, filter.Limit, filter.Offset)
	}

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*Incident
	for rows.Next() {
		var inc Incident
		var labelsJSON []byte
		err := rows.Scan(
			&inc.TenantID, &inc.IncidentID, &inc.CorrelatedGroupID, &inc.CreatedAt, &inc.UpdatedAt,
			&inc.Status, &inc.Severity, &inc.MaxRiskScore, &inc.MaxConfidence,
			&inc.MitreTactic, &inc.MitreTechnique, &labelsJSON, &inc.AssignedTo, &inc.ClosureReason, &inc.RequestID,
			&inc.AlertCount, &inc.HitCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		if err := json.Unmarshal(labelsJSON, &inc.Labels); err != nil {
			return nil, fmt.Errorf("unmarshal labels: %w", err)
		}
		incidents = append(incidents, &inc)
	}

	// Build next-page cursor from last row.
	result := &IncidentListResult{Incidents: incidents}
	if len(incidents) == filter.Limit {
		last := incidents[len(incidents)-1]
		result.NextCursor = last.UpdatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.IncidentID
	}
	return result, nil
}

func (s *IncidentStore) Update(ctx context.Context, tenantID, incidentID string, updates map[string]interface{}) error {
	allowed := map[string]bool{
		"status":         true,
		"assigned_to":    true,
		"closure_reason": true,
	}

	// Tenant ID and incident ID are positional args $1, $2.
	// Additional SET clause args start at $3.
	args := []interface{}{tenantID, incidentID}
	argPos := 3
	setClauses := []string{"updated_at = now()"}

	for key, val := range updates {
		if !allowed[key] {
			return fmt.Errorf("field %s not updatable", key)
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argPos))
		args = append(args, val)
		argPos++
	}

	// Build SET list first, then append the WHERE clause — never inject WHERE
	// in the middle of SET clauses (C3 fix: correct SQL construction).
	setList := setClauses[0]
	for i := 1; i < len(setClauses); i++ {
		setList += ", " + setClauses[i]
	}
	q := fmt.Sprintf("UPDATE incidents SET %s WHERE tenant_id = $1 AND incident_id = $2", setList)

	_, err := s.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update incident: %w", err)
	}
	return nil
}
