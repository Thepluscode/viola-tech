package normalizer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/viola/shared/kafka"
)

// CloudEvent is a provider-agnostic cloud event.
type CloudEvent struct {
	TenantID    string
	EntityID    string // resource ARN, subscription ID, etc.
	EventType   string // normalized event type
	ObservedAt  time.Time
	Source      string // aws-cloudtrail, azure-activity, gcp-audit
	Provider    string // aws, azure, gcp
	Region      string
	Principal   string // who performed the action
	Action      string // raw API action
	Resource    string // affected resource
	Result      string // success, failure
	RawPayload  json.RawMessage
	Labels      map[string]string
}

// Normalizer converts cloud events into Viola telemetry and publishes to Kafka.
type Normalizer struct {
	producer *kafka.Producer
	tenantID string
}

// New creates a normalizer.
func New(producer *kafka.Producer, tenantID string) *Normalizer {
	return &Normalizer{producer: producer, tenantID: tenantID}
}

// Publish normalizes and publishes a cloud event to Kafka.
func (n *Normalizer) Publish(ctx context.Context, event CloudEvent) error {
	// Build Viola telemetry envelope as JSON (will be protobuf in production)
	envelope := map[string]interface{}{
		"tenant_id":   event.TenantID,
		"entity_id":   event.EntityID,
		"observed_at":  event.ObservedAt.UTC().Format(time.RFC3339),
		"received_at":  time.Now().UTC().Format(time.RFC3339),
		"event_type":   event.EventType,
		"source":       event.Source,
		"payload":      event.RawPayload,
		"labels":       n.enrichLabels(event),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	headers := map[string]string{
		kafka.HdrTenantID:  event.TenantID,
		kafka.HdrRequestID: fmt.Sprintf("cloud-%s-%d", event.Provider, time.Now().UnixNano()),
		kafka.HdrSource:    event.Source,
		kafka.HdrSchema:    "viola.telemetry.v1.EventEnvelope",
		kafka.HdrEmittedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return n.producer.Produce(ctx, kafka.ProduceMessage{
		Key:     []byte(event.TenantID + ":" + event.EntityID),
		Value:   data,
		Headers: headers,
	})
}

func (n *Normalizer) enrichLabels(event CloudEvent) map[string]string {
	labels := map[string]string{
		"cloud_provider": event.Provider,
		"cloud_region":   event.Region,
		"principal":      event.Principal,
		"action":         event.Action,
		"result":         event.Result,
	}
	for k, v := range event.Labels {
		labels[k] = v
	}
	return labels
}
