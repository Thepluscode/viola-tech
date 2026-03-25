package materializer

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/viola/gateway-api/internal/store"
	alertv1 "github.com/viola/shared/proto/security"
)

type Materializer struct {
	incidentStore *store.IncidentStore
	alertStore    *store.AlertStore
}

func New(incidentStore *store.IncidentStore, alertStore *store.AlertStore) *Materializer {
	return &Materializer{
		incidentStore: incidentStore,
		alertStore:    alertStore,
	}
}

// HandleAlertCreated processes security.alert.v1.created events
func (m *Materializer) HandleAlertCreated(ctx context.Context, payload []byte) error {
	var event alertv1.Alert
	if err := proto.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal alert: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, event.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}
	updatedAt, err := time.Parse(time.RFC3339, event.UpdatedAt)
	if err != nil {
		updatedAt = time.Now()
	}

	alert := &store.Alert{
		TenantID:        event.TenantId,
		AlertID:         event.AlertId,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		Status:          event.Status,
		Severity:        event.Severity,
		Confidence:      event.Confidence,
		RiskScore:       event.RiskScore,
		Title:           event.Title,
		Description:     event.Description,
		MitreTactic:     ptrOrNil(event.MitreTactic),
		MitreTechnique:  ptrOrNil(event.MitreTechnique),
		Labels:          event.Labels,
		AssignedTo:      ptrOrNil(event.AssignedTo),
		ClosureReason:   ptrOrNil(event.ClosureReason),
		RequestID:       ptrOrNil(event.RequestId),
		EntityIDs:       event.EntityIds,
		DetectionHitIDs: event.DetectionHitIds,
	}

	if err := m.alertStore.Upsert(ctx, alert); err != nil {
		return fmt.Errorf("upsert alert: %w", err)
	}

	log.Printf("Materialized alert: %s (tenant: %s)", alert.AlertID, alert.TenantID)
	return nil
}

// HandleAlertUpdated processes security.alert.v1.updated events
func (m *Materializer) HandleAlertUpdated(ctx context.Context, payload []byte) error {
	// Same as created (upsert handles both)
	return m.HandleAlertCreated(ctx, payload)
}

// HandleIncidentUpserted processes security.incident.v1.upserted events
func (m *Materializer) HandleIncidentUpserted(ctx context.Context, payload []byte) error {
	var event alertv1.Incident
	if err := proto.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal incident: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, event.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}
	updatedAt, err := time.Parse(time.RFC3339, event.UpdatedAt)
	if err != nil {
		updatedAt = time.Now()
	}

	incident := &store.Incident{
		TenantID:          event.TenantId,
		IncidentID:        event.IncidentId,
		CorrelatedGroupID: event.CorrelatedGroupId,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		Status:            event.Status,
		Severity:          event.Severity,
		MaxRiskScore:      event.MaxRiskScore,
		MaxConfidence:     event.MaxConfidence,
		MitreTactic:       ptrOrNil(event.MitreTactic),
		MitreTechnique:    ptrOrNil(event.MitreTechnique),
		Labels:            event.Labels,
		AssignedTo:        ptrOrNil(event.AssignedTo),
		ClosureReason:     ptrOrNil(event.ClosureReason),
		RequestID:         ptrOrNil(event.RequestId),
		AlertCount:        int(event.AlertCount),
		HitCount:          int(event.HitCount),
		EntityIDs:         event.EntityIds,
		AlertIDs:          event.AlertIds,
		DetectionHitIDs:   event.DetectionHitIds,
	}

	if err := m.incidentStore.Upsert(ctx, incident); err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}

	log.Printf("Materialized incident: %s (tenant: %s, alerts: %d)", incident.IncidentID, incident.TenantID, incident.AlertCount)
	return nil
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
