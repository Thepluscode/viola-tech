package integration

import (
	"context"
	"testing"
	"time"

	"github.com/viola/shared/kafka"
	alertpb "github.com/viola/shared/proto/security"
	telemetrypb "github.com/viola/shared/proto/telemetry"
	"github.com/viola/tests/integration/testutil"
	"google.golang.org/protobuf/proto"
)

// TestPipeline_RawToNormalized verifies the ingestion service normalizes
// raw telemetry events and publishes them to the normalized topic.
func TestPipeline_RawToNormalized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.WaitForKafka(ctx, t)

	topics := kafka.NewTopics("dev")

	// Ensure topics exist.
	testutil.EnsureTopic(ctx, t, topics.TelemetryEndpointRaw, 1)
	testutil.EnsureTopic(ctx, t, topics.TelemetryNormalized, 1)

	// Produce a raw process_start event.
	env := testutil.NewProcessStartEnvelope("tenant-integ-001", "entity-001")
	testutil.ProduceRaw(ctx, t, topics.TelemetryEndpointRaw, env)

	// Consume from the normalized topic.
	msg := testutil.ConsumeOne(ctx, t, topics.TelemetryNormalized, "integ-pipeline-norm")

	// Verify the normalized message is a valid EventEnvelope.
	var normalized telemetrypb.EventEnvelope
	if err := proto.Unmarshal(msg.Value, &normalized); err != nil {
		t.Fatalf("unmarshal normalized event: %v", err)
	}

	if normalized.TenantId != "tenant-integ-001" {
		t.Errorf("tenant_id = %q, want %q", normalized.TenantId, "tenant-integ-001")
	}
	if normalized.EventType != "process_start" {
		t.Errorf("event_type = %q, want %q", normalized.EventType, "process_start")
	}
	if normalized.Source != "viola-agent" {
		t.Errorf("source = %q, want %q", normalized.Source, "viola-agent")
	}

	t.Logf("pipeline: raw → normalized OK (entity=%s)", normalized.EntityId)
}

// TestPipeline_NormalizedToAlert verifies the detection engine produces
// alerts from normalized telemetry events.
func TestPipeline_NormalizedToAlert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	testutil.WaitForKafka(ctx, t)

	topics := kafka.NewTopics("dev")

	testutil.EnsureTopic(ctx, t, topics.TelemetryEndpointRaw, 1)
	testutil.EnsureTopic(ctx, t, topics.AlertCreated, 1)

	// Produce a suspicious process event that should trigger detection.
	env := testutil.NewProcessStartEnvelope("tenant-integ-002", "entity-002")
	testutil.ProduceRaw(ctx, t, topics.TelemetryEndpointRaw, env)

	// Attempt to consume an alert. If detection doesn't fire for this event
	// type, the test will timeout — which is informative.
	alertCtx, alertCancel := context.WithTimeout(ctx, 20*time.Second)
	defer alertCancel()

	msg := testutil.ConsumeOne(alertCtx, t, topics.AlertCreated, "integ-pipeline-alert")

	var alert alertpb.Alert
	if err := proto.Unmarshal(msg.Value, &alert); err != nil {
		t.Fatalf("unmarshal alert: %v", err)
	}

	if alert.TenantId != "tenant-integ-002" {
		t.Errorf("alert tenant_id = %q, want %q", alert.TenantId, "tenant-integ-002")
	}
	if alert.Severity == "" {
		t.Error("alert severity is empty")
	}
	if alert.RiskScore <= 0 {
		t.Errorf("alert risk_score = %f, want > 0", alert.RiskScore)
	}

	t.Logf("pipeline: normalized → alert OK (id=%s, severity=%s, risk=%.2f)",
		alert.AlertId, alert.Severity, alert.RiskScore)
}

// TestPipeline_EventTypeNormalization verifies that the ingestion service
// canonicalizes non-standard event type names.
func TestPipeline_EventTypeNormalization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.WaitForKafka(ctx, t)

	topics := kafka.NewTopics("dev")
	testutil.EnsureTopic(ctx, t, topics.TelemetryEndpointRaw, 1)
	testutil.EnsureTopic(ctx, t, topics.TelemetryNormalized, 1)

	// Use non-canonical event type "proc_start" — ingestion should normalize to "process_start".
	env := testutil.NewProcessStartEnvelope("tenant-integ-003", "entity-003")
	env.EventType = "proc_start"
	testutil.ProduceRaw(ctx, t, topics.TelemetryEndpointRaw, env)

	msg := testutil.ConsumeOne(ctx, t, topics.TelemetryNormalized, "integ-pipeline-canonical")

	var normalized telemetrypb.EventEnvelope
	if err := proto.Unmarshal(msg.Value, &normalized); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if normalized.EventType != "process_start" {
		t.Errorf("event_type = %q, want %q (canonicalized)", normalized.EventType, "process_start")
	}

	t.Logf("pipeline: event type canonicalization OK (proc_start → %s)", normalized.EventType)
}
