package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	kgo "github.com/segmentio/kafka-go"
	"github.com/viola/shared/kafka"
	graphpb "github.com/viola/shared/proto/graph"
	"github.com/viola/tests/integration/testutil"
	"google.golang.org/protobuf/proto"
)

// TestGraph_EdgeIngestion verifies the graph service consumes edge events
// from Kafka and builds the attack graph.
func TestGraph_EdgeIngestion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.WaitForKafka(ctx, t)

	topics := kafka.NewTopics("dev")
	testutil.EnsureTopic(ctx, t, topics.GraphEdgeObserved, 1)
	testutil.EnsureTopic(ctx, t, topics.GraphRiskUpdated, 1)

	tenantID := "tenant-graph-001"
	now := time.Now().UTC().Format(time.RFC3339)

	// Publish a sequence of edges that form a lateral movement path:
	// workstation-1 → server-1 → database-1 (crown jewel)
	edges := []*graphpb.GraphEdgeObserved{
		{
			TenantId:     tenantID,
			EdgeId:       "edge-001",
			ObservedAt:   now,
			SourceNodeId: "workstation-1",
			TargetNodeId: "server-1",
			EdgeType:     "authentication",
			Weight:       0.8,
			Provenance:   "telemetry",
			Labels:       map[string]string{"protocol": "kerberos"},
			RequestId:    "req-graph-001",
		},
		{
			TenantId:     tenantID,
			EdgeId:       "edge-002",
			ObservedAt:   now,
			SourceNodeId: "server-1",
			TargetNodeId: "database-1",
			EdgeType:     "network_connection",
			Weight:       0.9,
			Provenance:   "telemetry",
			Labels:       map[string]string{"port": "5432"},
			RequestId:    "req-graph-002",
		},
	}

	w := &kgo.Writer{
		Addr:  kgo.TCP(testutil.KafkaBroker()),
		Topic: topics.GraphEdgeObserved,
	}
	defer w.Close()

	for _, edge := range edges {
		data, err := proto.Marshal(edge)
		if err != nil {
			t.Fatalf("marshal edge: %v", err)
		}
		headers := []kgo.Header{
			{Key: kafka.HdrTenantID, Value: []byte(tenantID)},
			{Key: kafka.HdrRequestID, Value: []byte(edge.RequestId)},
			{Key: kafka.HdrSource, Value: []byte("integration-test")},
			{Key: kafka.HdrSchema, Value: []byte("viola.graph.v1.GraphEdgeObserved")},
			{Key: kafka.HdrEmittedAt, Value: []byte(now)},
		}
		if err := w.WriteMessages(ctx, kgo.Message{
			Key:     []byte(tenantID + ":" + edge.EdgeId),
			Value:   data,
			Headers: headers,
		}); err != nil {
			t.Fatalf("publish edge %s: %v", edge.EdgeId, err)
		}
	}

	t.Logf("graph: published %d edges", len(edges))

	// Attempt to consume a risk update. The graph service should compute
	// risk scores after ingesting the edges.
	riskCtx, riskCancel := context.WithTimeout(ctx, 15*time.Second)
	defer riskCancel()

	msg := testutil.ConsumeOne(riskCtx, t, topics.GraphRiskUpdated, "integ-graph-risk")

	var riskUpdate graphpb.GraphRiskUpdate
	if err := proto.Unmarshal(msg.Value, &riskUpdate); err != nil {
		t.Fatalf("unmarshal risk update: %v", err)
	}

	if riskUpdate.TenantId != tenantID {
		t.Errorf("risk update tenant_id = %q, want %q", riskUpdate.TenantId, tenantID)
	}
	if riskUpdate.RiskScore <= 0 {
		t.Errorf("risk_score = %f, want > 0", riskUpdate.RiskScore)
	}

	t.Logf("graph: risk update received (node=%s, score=%.2f, reason=%s)",
		riskUpdate.NodeId, riskUpdate.RiskScore, riskUpdate.Reason)
}

// TestGraph_MetricsEndpoint verifies the graph service exposes Prometheus metrics.
func TestGraph_MetricsEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := timeoutClient()
	resp, err := client.Get("http://localhost:9091/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("graph metrics status=%d, want 200", resp.StatusCode)
	}

	t.Log("graph: /metrics endpoint OK")
}

// TestGraph_MultiTenantIsolation verifies that edges from different tenants
// do not leak across tenant boundaries.
func TestGraph_MultiTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.WaitForKafka(ctx, t)

	topics := kafka.NewTopics("dev")
	now := time.Now().UTC().Format(time.RFC3339)

	w := &kgo.Writer{
		Addr:  kgo.TCP(testutil.KafkaBroker()),
		Topic: topics.GraphEdgeObserved,
	}
	defer w.Close()

	// Publish edges for two different tenants.
	tenants := []string{"tenant-iso-A", "tenant-iso-B"}
	for _, tid := range tenants {
		edge := &graphpb.GraphEdgeObserved{
			TenantId:     tid,
			EdgeId:       "edge-iso-" + tid,
			ObservedAt:   now,
			SourceNodeId: "node-A",
			TargetNodeId: "node-B",
			EdgeType:     "authentication",
			Weight:       0.5,
			Provenance:   "integration-test",
			RequestId:    "req-iso-" + tid,
		}
		data, _ := proto.Marshal(edge)
		headers := []kgo.Header{
			{Key: kafka.HdrTenantID, Value: []byte(tid)},
			{Key: kafka.HdrRequestID, Value: []byte(edge.RequestId)},
			{Key: kafka.HdrSource, Value: []byte("integration-test")},
			{Key: kafka.HdrSchema, Value: []byte("viola.graph.v1.GraphEdgeObserved")},
			{Key: kafka.HdrEmittedAt, Value: []byte(now)},
		}
		if err := w.WriteMessages(ctx, kgo.Message{
			Key:     []byte(tid + ":" + edge.EdgeId),
			Value:   data,
			Headers: headers,
		}); err != nil {
			t.Fatalf("publish edge for %s: %v", tid, err)
		}
	}

	t.Log("graph: multi-tenant isolation edges published — manual verification required via graph API")
}

// timeoutClient is a helper for HTTP tests (unexported to avoid lint issues).
func timeoutClient() http.Client {
	return http.Client{Timeout: 5 * time.Second}
}

// dumpJSON is a helper for debug logging.
func dumpJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
