package audit

import (
	"context"
	"encoding/json"
	"time"

	sharedkafka "github.com/viola/shared/kafka"
	auditv1 "github.com/viola/shared/proto/audit"
	"github.com/viola/shared/id"
)

// Emitter publishes audit events to Kafka
type Emitter struct {
	service string
	prod    *sharedkafka.Producer
	topic   string
}

// Config configures the audit emitter
type Config struct {
	Service string
	Brokers []string
	Topic   string
}

// New creates a new audit emitter
func New(cfg Config) (*Emitter, error) {
	p, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
	})
	if err != nil {
		return nil, err
	}
	return &Emitter{service: cfg.Service, prod: p, topic: cfg.Topic}, nil
}

// Close closes the audit emitter
func (e *Emitter) Close() error { return e.prod.Close() }

// Emit publishes an audit event
func (e *Emitter) Emit(ctx context.Context, tenantID, requestID string, ev *auditv1.AuditEvent) error {
	now := time.Now().UTC()
	ev.AuditId = id.New()
	ev.OccurredAt = now.Format(time.RFC3339)
	ev.TenantId = tenantID
	ev.RequestId = requestID

	// For MVP, use JSON encoding (can switch to protobuf later)
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	headers := map[string]string{
		sharedkafka.HdrTenantID:  tenantID,
		sharedkafka.HdrRequestID: requestID,
		sharedkafka.HdrSource:    e.service,
		sharedkafka.HdrSchema:    "viola.audit.v1.AuditEvent",
		sharedkafka.HdrEmittedAt: now.Format(time.RFC3339),
	}
	key := []byte(tenantID + ":" + ev.AuditId)
	return e.prod.Produce(ctx, sharedkafka.ProduceMessage{Key: key, Value: b, Headers: headers})
}
