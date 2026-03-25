package incident

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	securityv1 "github.com/viola/shared/proto/security"
)

type incidentUpsertInput struct {
	tenantID       string
	incidentID     string
	groupID        string
	createdAt      time.Time
	updatedAt      time.Time
	status         string
	severity       string
	maxRiskScore   float64
	maxConfidence  float64
	mitreTactic    string
	mitreTechnique string
	labels         map[string]string
	requestID      string
}

// upsertIncident writes or merges an incident row using a single atomic statement.
//
// Correctness guarantees:
//   - Severity is merged via CASE WHEN in SQL — no extra round-trip, no application lock.
//     The comparison uses an inline ranking expression identical to the former Go map so
//     the higher severity always wins without re-reading the existing row.
//   - max_risk_score and max_confidence use GREATEST (safe, built-in).
//   - alert_count is NOT managed here; it is recomputed from the link table in
//     linkIncidentAlert to prevent double-counting under Kafka replay.
//   - created_at is preserved on conflict (DO NOT overwrite).
//   - Incident status is intentionally not overwritten by incoming alert status;
//     status is managed by SOC workflow, not by detection events.
func upsertIncident(ctx context.Context, pool *pgxpool.Pool, in incidentUpsertInput) error {
	lbl, _ := json.Marshal(in.labels)

	// Single atomic INSERT ... ON CONFLICT DO UPDATE.
	// Severity merge uses a CASE WHEN ranking that mirrors Go severityRank:
	//   critical=4, high=3, med=2, low=1 — the higher rank wins.
	_, err := pool.Exec(ctx, `
INSERT INTO incidents (
  tenant_id, incident_id, correlated_group_id,
  created_at, updated_at, status,
  severity, max_risk_score, max_confidence,
  mitre_tactic, mitre_technique, labels,
  request_id, alert_count, hit_count
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,0,0)
ON CONFLICT (tenant_id, incident_id) DO UPDATE SET
  updated_at      = EXCLUDED.updated_at,
  severity        = CASE
    WHEN (CASE incidents.severity
            WHEN 'critical' THEN 4
            WHEN 'high'     THEN 3
            WHEN 'med'      THEN 2
            ELSE 1 END)
       >= (CASE EXCLUDED.severity
            WHEN 'critical' THEN 4
            WHEN 'high'     THEN 3
            WHEN 'med'      THEN 2
            ELSE 1 END)
    THEN incidents.severity
    ELSE EXCLUDED.severity
  END,
  max_risk_score  = GREATEST(incidents.max_risk_score,  EXCLUDED.max_risk_score),
  max_confidence  = GREATEST(incidents.max_confidence,  EXCLUDED.max_confidence),
  mitre_tactic    = COALESCE(EXCLUDED.mitre_tactic,    incidents.mitre_tactic),
  mitre_technique = COALESCE(EXCLUDED.mitre_technique, incidents.mitre_technique),
  labels          = incidents.labels || EXCLUDED.labels,
  request_id      = EXCLUDED.request_id
`,
		in.tenantID, in.incidentID, in.groupID,
		in.createdAt, in.updatedAt, in.status,
		in.severity, in.maxRiskScore, in.maxConfidence,
		nullIfEmpty(in.mitreTactic), nullIfEmpty(in.mitreTechnique), lbl,
		in.requestID,
	)
	return err
}

// linkIncidentEntity inserts a single entity association idempotently.
func linkIncidentEntity(ctx context.Context, pool *pgxpool.Pool, tenantID, incidentID, entityID string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO incident_entities (tenant_id, incident_id, entity_id)
VALUES ($1,$2,$3)
ON CONFLICT DO NOTHING
`, tenantID, incidentID, entityID)
	return err
}

// linkIncidentEntitiesBatch inserts multiple entity associations in a single
// round-trip using UNNEST. This replaces N individual INSERT calls with one
// query, reducing per-alert DB overhead from O(N) to O(1) network RTTs.
//
// Any entity IDs already linked are silently skipped (ON CONFLICT DO NOTHING).
func linkIncidentEntitiesBatch(ctx context.Context, pool *pgxpool.Pool, tenantID, incidentID string, entityIDs []string) error {
	if len(entityIDs) == 0 {
		return nil
	}
	if len(entityIDs) == 1 {
		return linkIncidentEntity(ctx, pool, tenantID, incidentID, entityIDs[0])
	}

	// Build parallel arrays for UNNEST: tenant and incident are repeated constants.
	tenantIDs := make([]string, len(entityIDs))
	incidentIDs := make([]string, len(entityIDs))
	for i := range entityIDs {
		tenantIDs[i] = tenantID
		incidentIDs[i] = incidentID
	}

	_, err := pool.Exec(ctx, `
INSERT INTO incident_entities (tenant_id, incident_id, entity_id)
SELECT * FROM UNNEST($1::text[], $2::text[], $3::text[])
ON CONFLICT DO NOTHING
`, tenantIDs, incidentIDs, entityIDs)
	return err
}

// linkIncidentAlert inserts an alert association idempotently, then recomputes
// alert_count from the link table so Kafka replays do not inflate the counter.
func linkIncidentAlert(ctx context.Context, pool *pgxpool.Pool, tenantID, incidentID, alertID string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO incident_alerts (tenant_id, incident_id, alert_id)
VALUES ($1,$2,$3)
ON CONFLICT DO NOTHING
`, tenantID, incidentID, alertID)
	if err != nil {
		return err
	}

	// Recompute from link table — safe under replay because inserts are idempotent.
	_, err = pool.Exec(ctx, `
UPDATE incidents
SET
  alert_count = (SELECT COUNT(*) FROM incident_alerts WHERE tenant_id=$1 AND incident_id=$2),
  updated_at  = NOW()
WHERE tenant_id=$1 AND incident_id=$2
`, tenantID, incidentID)
	return err
}

// readIncidentSnapshot fetches the current aggregated state of an incident.
// Returns a proto-ready struct. entity_ids and alert_ids are intentionally
// omitted here for performance; fetch them separately when needed by the UI.
func readIncidentSnapshot(ctx context.Context, pool *pgxpool.Pool, tenantID, incidentID string) (*securityv1.Incident, error) {
	row := pool.QueryRow(ctx, `
SELECT
  tenant_id, incident_id, correlated_group_id,
  created_at, updated_at, status,
  severity, max_risk_score, max_confidence,
  COALESCE(mitre_tactic,''), COALESCE(mitre_technique,''),
  labels,
  COALESCE(assigned_to,''), COALESCE(closure_reason,''), COALESCE(request_id,''),
  alert_count, hit_count
FROM incidents
WHERE tenant_id=$1 AND incident_id=$2
`, tenantID, incidentID)

	var (
		tid, iid, gid                              string
		status, sev, tactic, technique             string
		assignedTo, closureReason, requestID       string
		createdAt, updatedAt                       time.Time
		labelsJSON                                 []byte
		maxRisk, maxConf                           float64
		alertCount, hitCount                       int32
	)

	if err := row.Scan(
		&tid, &iid, &gid,
		&createdAt, &updatedAt, &status,
		&sev, &maxRisk, &maxConf,
		&tactic, &technique,
		&labelsJSON,
		&assignedTo, &closureReason, &requestID,
		&alertCount, &hitCount,
	); err != nil {
		return nil, err
	}

	labels := map[string]string{}
	_ = json.Unmarshal(labelsJSON, &labels)

	return &securityv1.Incident{
		TenantId:          tid,
		IncidentId:        iid,
		CorrelatedGroupId: gid,
		CreatedAt:         createdAt.UTC().Format(time.RFC3339),
		UpdatedAt:         updatedAt.UTC().Format(time.RFC3339),
		Status:            status,
		Severity:          sev,
		MaxRiskScore:      maxRisk,
		MaxConfidence:     maxConf,
		MitreTactic:       tactic,
		MitreTechnique:    technique,
		Labels:            labels,
		AssignedTo:        assignedTo,
		ClosureReason:     closureReason,
		RequestId:         requestID,
		AlertCount:        alertCount,
		HitCount:          hitCount,
	}, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
