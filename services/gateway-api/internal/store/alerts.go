package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Alert struct {
	TenantID        string            `json:"tenant_id"`
	AlertID         string            `json:"alert_id"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Status          string            `json:"status"`
	Severity        string            `json:"severity"`
	Confidence      float64           `json:"confidence"`
	RiskScore       float64           `json:"risk_score"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	MitreTactic     *string           `json:"mitre_tactic,omitempty"`
	MitreTechnique  *string           `json:"mitre_technique,omitempty"`
	Labels          map[string]string `json:"labels"`
	AssignedTo      *string           `json:"assigned_to,omitempty"`
	ClosureReason   *string           `json:"closure_reason,omitempty"`
	RequestID       *string           `json:"request_id,omitempty"`
	EntityIDs       []string          `json:"entity_ids"`
	DetectionHitIDs []string          `json:"detection_hit_ids"`
}

type AlertStore struct {
	pool *pgxpool.Pool
}

func NewAlertStore(pool *pgxpool.Pool) *AlertStore {
	return &AlertStore{pool: pool}
}

func (s *AlertStore) Upsert(ctx context.Context, alert *Alert) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	labelsJSON, err := json.Marshal(alert.Labels)
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	// Upsert main alert record
	q := `
		INSERT INTO alerts (
			tenant_id, alert_id, created_at, updated_at, status,
			severity, confidence, risk_score, title, description,
			mitre_tactic, mitre_technique, labels, assigned_to, closure_reason, request_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (tenant_id, alert_id) DO UPDATE SET
			updated_at = EXCLUDED.updated_at,
			status = EXCLUDED.status,
			severity = EXCLUDED.severity,
			confidence = EXCLUDED.confidence,
			risk_score = EXCLUDED.risk_score,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			mitre_tactic = EXCLUDED.mitre_tactic,
			mitre_technique = EXCLUDED.mitre_technique,
			labels = EXCLUDED.labels,
			assigned_to = EXCLUDED.assigned_to,
			closure_reason = EXCLUDED.closure_reason
	`
	_, err = tx.Exec(ctx, q,
		alert.TenantID, alert.AlertID, alert.CreatedAt, alert.UpdatedAt, alert.Status,
		alert.Severity, alert.Confidence, alert.RiskScore, alert.Title, alert.Description,
		alert.MitreTactic, alert.MitreTechnique, labelsJSON, alert.AssignedTo, alert.ClosureReason, alert.RequestID,
	)
	if err != nil {
		return fmt.Errorf("upsert alert: %w", err)
	}

	// Upsert entity associations
	if len(alert.EntityIDs) > 0 {
		for _, entityID := range alert.EntityIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO alert_entities (tenant_id, alert_id, entity_id)
				VALUES ($1, $2, $3)
				ON CONFLICT (tenant_id, alert_id, entity_id) DO NOTHING
			`, alert.TenantID, alert.AlertID, entityID)
			if err != nil {
				return fmt.Errorf("upsert entity: %w", err)
			}
		}
	}

	// Upsert hit associations
	if len(alert.DetectionHitIDs) > 0 {
		for _, hitID := range alert.DetectionHitIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO alert_hits (tenant_id, alert_id, hit_id)
				VALUES ($1, $2, $3)
				ON CONFLICT (tenant_id, alert_id, hit_id) DO NOTHING
			`, alert.TenantID, alert.AlertID, hitID)
			if err != nil {
				return fmt.Errorf("upsert hit: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *AlertStore) Get(ctx context.Context, tenantID, alertID string) (*Alert, error) {
	q := `
		SELECT tenant_id, alert_id, created_at, updated_at, status,
			severity, confidence, risk_score, title, description,
			mitre_tactic, mitre_technique, labels, assigned_to, closure_reason, request_id
		FROM alerts
		WHERE tenant_id = $1 AND alert_id = $2
	`
	var alert Alert
	var labelsJSON []byte
	err := s.pool.QueryRow(ctx, q, tenantID, alertID).Scan(
		&alert.TenantID, &alert.AlertID, &alert.CreatedAt, &alert.UpdatedAt, &alert.Status,
		&alert.Severity, &alert.Confidence, &alert.RiskScore, &alert.Title, &alert.Description,
		&alert.MitreTactic, &alert.MitreTechnique, &labelsJSON, &alert.AssignedTo, &alert.ClosureReason, &alert.RequestID,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query alert: %w", err)
	}

	if err := json.Unmarshal(labelsJSON, &alert.Labels); err != nil {
		return nil, fmt.Errorf("unmarshal labels: %w", err)
	}

	// Load entities
	rows, err := s.pool.Query(ctx, `SELECT entity_id FROM alert_entities WHERE tenant_id = $1 AND alert_id = $2`, tenantID, alertID)
	if err != nil {
		return nil, fmt.Errorf("query entities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var eid string
		if err := rows.Scan(&eid); err != nil {
			return nil, err
		}
		alert.EntityIDs = append(alert.EntityIDs, eid)
	}

	// Load hits
	rows, err = s.pool.Query(ctx, `SELECT hit_id FROM alert_hits WHERE tenant_id = $1 AND alert_id = $2`, tenantID, alertID)
	if err != nil {
		return nil, fmt.Errorf("query hits: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var hid string
		if err := rows.Scan(&hid); err != nil {
			return nil, err
		}
		alert.DetectionHitIDs = append(alert.DetectionHitIDs, hid)
	}

	return &alert, nil
}

// ListAlertsFilter controls which alerts are returned.
// Cursor-based pagination is preferred over offset:
//   - Set AfterUpdatedAt + AfterID to the last row's (updated_at, alert_id)
//     from the previous page to fetch the next page in O(log N) index seeks.
//   - Fall back to Offset when cursor fields are zero (backwards-compatible).
type ListAlertsFilter struct {
	Status   string
	Severity string
	Limit    int

	// Cursor pagination (keyset): supply both fields from the previous page's last row.
	AfterUpdatedAt time.Time
	AfterID        string

	// Legacy offset pagination (avoid for large tables).
	Offset int
}

// AlertListResult wraps page rows and an opaque next-page cursor.
type AlertListResult struct {
	Alerts     []*Alert
	NextCursor string // "<rfc3339>|<alert_id>" — empty when no next page
}

func (s *AlertStore) List(ctx context.Context, tenantID string, filter ListAlertsFilter) (*AlertListResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	q := `
		SELECT tenant_id, alert_id, created_at, updated_at, status,
			severity, confidence, risk_score, title, description,
			mitre_tactic, mitre_technique, labels, assigned_to, closure_reason, request_id
		FROM alerts
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
		q += fmt.Sprintf(
			" AND (updated_at, alert_id) < ($%d, $%d)",
			argPos, argPos+1,
		)
		args = append(args, filter.AfterUpdatedAt, filter.AfterID)
		argPos += 2
		q += fmt.Sprintf(" ORDER BY updated_at DESC, alert_id DESC LIMIT $%d", argPos)
		args = append(args, filter.Limit)
	} else {
		q += fmt.Sprintf(" ORDER BY updated_at DESC, alert_id DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
		args = append(args, filter.Limit, filter.Offset)
	}

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var alert Alert
		var labelsJSON []byte
		err := rows.Scan(
			&alert.TenantID, &alert.AlertID, &alert.CreatedAt, &alert.UpdatedAt, &alert.Status,
			&alert.Severity, &alert.Confidence, &alert.RiskScore, &alert.Title, &alert.Description,
			&alert.MitreTactic, &alert.MitreTechnique, &labelsJSON, &alert.AssignedTo, &alert.ClosureReason, &alert.RequestID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		if err := json.Unmarshal(labelsJSON, &alert.Labels); err != nil {
			return nil, fmt.Errorf("unmarshal labels: %w", err)
		}
		alerts = append(alerts, &alert)
	}

	result := &AlertListResult{Alerts: alerts}
	if len(alerts) == filter.Limit {
		last := alerts[len(alerts)-1]
		result.NextCursor = last.UpdatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.AlertID
	}
	return result, nil
}

func (s *AlertStore) Update(ctx context.Context, tenantID, alertID string, updates map[string]interface{}) error {
	allowed := map[string]bool{
		"status":         true,
		"assigned_to":    true,
		"closure_reason": true,
	}

	// Tenant ID and alert ID are positional args $1, $2.
	// Additional SET clause args start at $3.
	args := []interface{}{tenantID, alertID}
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
	q := fmt.Sprintf("UPDATE alerts SET %s WHERE tenant_id = $1 AND alert_id = $2", setList)

	_, err := s.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update alert: %w", err)
	}
	return nil
}
