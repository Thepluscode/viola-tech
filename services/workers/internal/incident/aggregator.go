package incident

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	sharedkafka "github.com/viola/shared/kafka"
	securityv1 "github.com/viola/shared/proto/security"
)

// Aggregator consumes alert.created events and materialises incidents.
type Aggregator struct {
	pool   *pgxpool.Pool
	out    *sharedkafka.Producer
	topics sharedkafka.Topics
}

// Config is the constructor configuration for Aggregator.
type Config struct {
	Pool    *pgxpool.Pool
	Env     string
	Brokers []string
}

// New creates an Aggregator and its outbound Kafka producer.
func New(cfg Config) (*Aggregator, error) {
	topics := sharedkafka.NewTopics(cfg.Env)
	out, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{
		Brokers: cfg.Brokers,
		Topic:   topics.IncidentUpserted,
	})
	if err != nil {
		return nil, err
	}
	return &Aggregator{pool: cfg.Pool, out: out, topics: topics}, nil
}

// Close releases the outbound producer.
func (a *Aggregator) Close() error { return a.out.Close() }

// HandleAlertCreated processes a single alert.created message:
//  1. Upserts the parent incident (severity merged in Go, not via UDF).
//  2. Links the alert and its entities to the incident.
//  3. Reads the current incident snapshot.
//  4. Publishes an incident.upserted event downstream.
func (a *Aggregator) HandleAlertCreated(ctx context.Context, msg sharedkafka.Message) error {
	codec := sharedkafka.ProtobufCodec[*securityv1.Alert]{
		Schema: "viola.security.v1.Alert",
		New:    func() *securityv1.Alert { return &securityv1.Alert{} },
	}
	alert, err := codec.Decode(msg.Value)
	if err != nil {
		return fmt.Errorf("decode alert: %w", err)
	}
	if alert.TenantId == "" || alert.TenantId != msg.TenantID {
		return fmt.Errorf("tenant mismatch: envelope=%s proto=%s", msg.TenantID, alert.TenantId)
	}

	// Resolve the correlation group ID.
	groupID := alert.CorrelatedGroupId
	if groupID == "" && alert.Labels != nil {
		groupID = alert.Labels["group_id"]
	}
	if groupID == "" {
		return fmt.Errorf("alert %s missing correlated_group_id", alert.AlertId)
	}

	// Use the group ID as the stable incident ID within a tenant.
	incidentID := groupID
	now := time.Now().UTC()

	// Step 1 — upsert incident (severity resolved in Go, no UDF).
	if err := upsertIncident(ctx, a.pool, incidentUpsertInput{
		tenantID:       alert.TenantId,
		incidentID:     incidentID,
		groupID:        groupID,
		createdAt:      now,
		updatedAt:      now,
		status:         "open",
		severity:       alert.Severity,
		maxRiskScore:   alert.RiskScore,
		maxConfidence:  alert.Confidence,
		mitreTactic:    alert.MitreTactic,
		mitreTechnique: alert.MitreTechnique,
		labels:         alert.Labels,
		requestID:      msg.RequestID,
	}); err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}

	// Step 2 — link alert (idempotent; alert_count recomputed from link table).
	if err := linkIncidentAlert(ctx, a.pool, alert.TenantId, incidentID, alert.AlertId); err != nil {
		return fmt.Errorf("link alert: %w", err)
	}

	// Step 3 — link entities in a single batch round-trip via UNNEST.
	var validEntityIDs []string
	for _, eid := range alert.EntityIds {
		if eid != "" {
			validEntityIDs = append(validEntityIDs, eid)
		}
	}
	if err := linkIncidentEntitiesBatch(ctx, a.pool, alert.TenantId, incidentID, validEntityIDs); err != nil {
		return fmt.Errorf("link entities: %w", err)
	}

	// Step 4 — read current snapshot and publish downstream.
	inc, err := readIncidentSnapshot(ctx, a.pool, alert.TenantId, incidentID)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	inc.RequestId = msg.RequestID
	inc.UpdatedAt = now.Format(time.RFC3339)

	incCodec := sharedkafka.ProtobufCodec[*securityv1.Incident]{
		Schema: "viola.security.v1.Incident",
		New:    func() *securityv1.Incident { return &securityv1.Incident{} },
	}
	b, err := incCodec.Encode(inc)
	if err != nil {
		return fmt.Errorf("encode incident: %w", err)
	}

	headers := map[string]string{
		sharedkafka.HdrTenantID:  alert.TenantId,
		sharedkafka.HdrRequestID: msg.RequestID,
		sharedkafka.HdrSource:    "workers",
		sharedkafka.HdrSchema:    incCodec.SchemaName(),
		sharedkafka.HdrEmittedAt: now.Format(time.RFC3339),
	}
	key := []byte(fmt.Sprintf("%s:%s", alert.TenantId, incidentID))
	return a.out.Produce(ctx, sharedkafka.ProduceMessage{Key: key, Value: b, Headers: headers})
}
