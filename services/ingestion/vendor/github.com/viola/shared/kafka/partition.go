package kafka

import (
	"fmt"
)

// PartitionKeyStrategy defines how to compute partition keys for different topics.
// This ensures:
// - Event ordering per entity (same partition for same key)
// - Tenant isolation (tenant data groups together)
// - Load distribution (avoids hot partitions where possible)
type PartitionKeyStrategy struct {
	env string
}

func NewPartitionKeyStrategy(env string) *PartitionKeyStrategy {
	return &PartitionKeyStrategy{env: env}
}

// KeyForTopic returns the partition key for a given topic and message metadata.
// Falls back to tenant_id if specific strategy is not defined.
func (s *PartitionKeyStrategy) KeyForTopic(topic string, tenantID string, entityID string, objectID string) []byte {
	// Telemetry topics: partition by entity_id for better distribution
	if topic == fmt.Sprintf("viola.%s.telemetry.endpoint.v1.raw", s.env) ||
		topic == fmt.Sprintf("viola.%s.telemetry.identity.v1.raw", s.env) ||
		topic == fmt.Sprintf("viola.%s.telemetry.cloud.v1.raw", s.env) ||
		topic == fmt.Sprintf("viola.%s.telemetry.v1.normalized", s.env) {
		if entityID != "" {
			return []byte(entityID)
		}
	}

	// Security topics: partition by entity_id for correlation
	if topic == fmt.Sprintf("viola.%s.security.detection.v1.hit", s.env) {
		if entityID != "" {
			return []byte(entityID)
		}
	}

	// Alert/incident lifecycle: partition by alert_id/incident_id for ordering
	if topic == fmt.Sprintf("viola.%s.security.alert.v1.created", s.env) ||
		topic == fmt.Sprintf("viola.%s.security.alert.v1.updated", s.env) {
		if objectID != "" {
			return []byte(objectID) // alert_id
		}
	}

	if topic == fmt.Sprintf("viola.%s.security.incident.v1.upserted", s.env) {
		if objectID != "" {
			return []byte(objectID) // incident_id
		}
	}

	// Graph topics: partition by source_node_id for graph build ordering
	if topic == fmt.Sprintf("viola.%s.graph.v1.edge.observed", s.env) {
		if entityID != "" {
			return []byte(entityID) // source_node_id
		}
	}

	if topic == fmt.Sprintf("viola.%s.graph.v1.risk.updated", s.env) {
		if entityID != "" {
			return []byte(entityID) // node_id
		}
	}

	// Response topics: partition by response_id for lifecycle ordering
	if topic == fmt.Sprintf("viola.%s.response.v1.requested", s.env) ||
		topic == fmt.Sprintf("viola.%s.response.v1.executed", s.env) ||
		topic == fmt.Sprintf("viola.%s.response.v1.failed", s.env) {
		if objectID != "" {
			return []byte(objectID) // response_id
		}
	}

	// Audit and DLQ: partition by tenant_id for isolation
	// (Also fallback for any topic without specific strategy)
	return []byte(tenantID)
}

// KeyForAlert is a convenience method for alert messages
func (s *PartitionKeyStrategy) KeyForAlert(tenantID string, alertID string) []byte {
	return []byte(alertID)
}

// KeyForIncident is a convenience method for incident messages
func (s *PartitionKeyStrategy) KeyForIncident(tenantID string, incidentID string) []byte {
	return []byte(incidentID)
}

// KeyForDetectionHit is a convenience method for detection hit messages
func (s *PartitionKeyStrategy) KeyForDetectionHit(tenantID string, entityID string) []byte {
	if entityID != "" {
		return []byte(entityID)
	}
	return []byte(tenantID)
}

// KeyForGraphEdge is a convenience method for graph edge messages
func (s *PartitionKeyStrategy) KeyForGraphEdge(tenantID string, sourceNodeID string) []byte {
	if sourceNodeID != "" {
		return []byte(sourceNodeID)
	}
	return []byte(tenantID)
}

// KeyForTenant is a convenience method for tenant-partitioned messages
func (s *PartitionKeyStrategy) KeyForTenant(tenantID string) []byte {
	return []byte(tenantID)
}
