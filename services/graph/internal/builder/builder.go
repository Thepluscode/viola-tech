package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/viola/graph/internal/store"
	sharedkafka "github.com/viola/shared/kafka"
	"github.com/viola/shared/observability/logging"
	"github.com/viola/shared/observability/metrics"
	"github.com/viola/shared/observability/tracing"
	telemetryv1 "github.com/viola/shared/proto/telemetry"
)

// Builder builds the attack graph from telemetry events
type Builder struct {
	manager *store.GraphManager
	metrics *metrics.Registry
	logger  *logging.Logger
	tracer  trace.Tracer
}

func New(manager *store.GraphManager) *Builder {
	return &Builder{
		manager: manager,
		metrics: metrics.NewRegistry("graph"),
		logger:  logging.New("graph", logging.INFO),
		tracer:  tracing.Tracer("graph"),
	}
}

// HandleTelemetryEvent processes a single telemetry event and updates the graph
func (b *Builder) HandleTelemetryEvent(ctx context.Context, msg sharedkafka.Message) error {
	start := time.Now()

	// Add request context for logging
	ctx = logging.WithContext(ctx, msg.RequestID, msg.TenantID)

	// Start tracing span
	ctx, span := tracing.StartSpan(ctx, b.tracer, "HandleTelemetryEvent",
		tracing.TenantID(msg.TenantID),
		tracing.RequestID(msg.RequestID),
	)
	defer span.End()

	// Decode telemetry event
	envCodec := sharedkafka.ProtobufCodec[*telemetryv1.EventEnvelope]{
		Schema: "viola.telemetry.v1.EventEnvelope",
		New:    func() *telemetryv1.EventEnvelope { return &telemetryv1.EventEnvelope{} },
	}
	ev, err := envCodec.Decode(msg.Value)
	if err != nil {
		b.metrics.EventsErrors.WithLabelValues(msg.TenantID, "decode_error").Inc()
		tracing.RecordError(ctx, err)
		return fmt.Errorf("decode envelope: %w", err)
	}

	if ev.TenantId == "" || ev.TenantId != msg.TenantID {
		b.metrics.EventsErrors.WithLabelValues(msg.TenantID, "tenant_mismatch").Inc()
		return fmt.Errorf("tenant mismatch")
	}

	// Track event processed
	b.metrics.EventsProcessed.WithLabelValues(ev.TenantId, ev.EventType).Inc()

	// Parse payload to extract fields
	fields, err := parsePayload(ev.Payload)
	if err != nil {
		b.logger.Warnf(ctx, "failed to parse payload for %s: %v", ev.EventType, err)
		fields = make(map[string]interface{})
	}

	// Extract relationships based on event type
	var handleErr error
	switch ev.EventType {
	case "process_start":
		handleErr = b.handleProcessStart(ctx, ev, fields)
	case "network_connect":
		handleErr = b.handleNetworkConnect(ctx, ev, fields)
	case "authentication_success":
		handleErr = b.handleAuthSuccess(ctx, ev, fields)
	default:
		// Ignore other event types for now
		return nil
	}

	// Track processing time
	duration := time.Since(start).Seconds()
	b.metrics.ProcessingTime.WithLabelValues(ev.TenantId, ev.EventType).Observe(duration)

	if handleErr != nil {
		b.metrics.EventsErrors.WithLabelValues(ev.TenantId, ev.EventType).Inc()
		tracing.RecordError(ctx, handleErr)
	}

	return handleErr
}

func (b *Builder) handleProcessStart(ctx context.Context, ev *telemetryv1.EventEnvelope, fields map[string]interface{}) error {
	ctx, span := tracing.StartSpan(ctx, b.tracer, "handleProcessStart",
		tracing.EventType("process_start"),
		tracing.EntityID(ev.EntityId),
	)
	defer span.End()
	// Add endpoint node
	endpointNode := &store.Node{
		ID:          fmt.Sprintf("endpoint:%s", ev.EntityId),
		Type:        store.NodeTypeEndpoint,
		Labels:      map[string]string{},
		Criticality: 0, // Default, can be overridden by crown jewel config
	}
	if err := b.manager.AddNode(ev.TenantId, endpointNode); err != nil {
		return fmt.Errorf("add endpoint node: %w", err)
	}

	// Extract user (if available)
	user, _ := fields["user"].(string)
	if user != "" {
		// Add user node
		userNode := &store.Node{
			ID:          fmt.Sprintf("user:%s", user),
			Type:        store.NodeTypeUser,
			Labels:      map[string]string{},
			Criticality: 0,
		}
		if err := b.manager.AddNode(ev.TenantId, userNode); err != nil {
			return fmt.Errorf("add user node: %w", err)
		}

		// Add spawn edge (user spawned process on endpoint)
		spawnEdge := &store.Edge{
			ID:       fmt.Sprintf("spawn:%s:%s:%d", user, ev.EntityId, time.Now().Unix()),
			Type:     store.EdgeTypeSpawn,
			Source:   fmt.Sprintf("user:%s", user),
			Target:   fmt.Sprintf("endpoint:%s", ev.EntityId),
			Weight:   1.0,
			TTL:      1 * time.Hour, // Spawn relationships expire after 1 hour
			Metadata: map[string]string{"event_type": "process_start"},
		}
		if err := b.manager.AddEdge(ev.TenantId, spawnEdge); err != nil {
			return fmt.Errorf("add spawn edge: %w", err)
		}
	}

	return nil
}

func (b *Builder) handleNetworkConnect(ctx context.Context, ev *telemetryv1.EventEnvelope, fields map[string]interface{}) error {
	ctx, span := tracing.StartSpan(ctx, b.tracer, "handleNetworkConnect",
		tracing.EventType("network_connect"),
		tracing.EntityID(ev.EntityId),
	)
	defer span.End()
	// Source endpoint
	sourceNode := &store.Node{
		ID:          fmt.Sprintf("endpoint:%s", ev.EntityId),
		Type:        store.NodeTypeEndpoint,
		Labels:      map[string]string{},
		Criticality: 0,
	}
	if err := b.manager.AddNode(ev.TenantId, sourceNode); err != nil {
		return fmt.Errorf("add source node: %w", err)
	}

	// Extract destination IP
	destIP, _ := fields["dest_ip"].(string)
	if destIP == "" {
		return nil // No destination, skip
	}

	// For MVP, we don't have a mapping of IP → endpoint ID
	// In production, you'd maintain an IP → endpoint mapping
	// For now, create a node based on IP
	destNode := &store.Node{
		ID:          fmt.Sprintf("endpoint:%s", destIP),
		Type:        store.NodeTypeEndpoint,
		Labels:      map[string]string{"ip": destIP},
		Criticality: 0,
	}
	if err := b.manager.AddNode(ev.TenantId, destNode); err != nil {
		return fmt.Errorf("add dest node: %w", err)
	}

	// Add network edge
	port, _ := fields["dest_port"].(string)
	networkEdge := &store.Edge{
		ID:       fmt.Sprintf("network:%s:%s:%d", ev.EntityId, destIP, time.Now().Unix()),
		Type:     store.EdgeTypeNetwork,
		Source:   fmt.Sprintf("endpoint:%s", ev.EntityId),
		Target:   fmt.Sprintf("endpoint:%s", destIP),
		Weight:   1.0,
		TTL:      30 * time.Minute, // Network connections expire after 30 min
		Metadata: map[string]string{"port": port},
	}
	if err := b.manager.AddEdge(ev.TenantId, networkEdge); err != nil {
		return fmt.Errorf("add network edge: %w", err)
	}

	return nil
}

func (b *Builder) handleAuthSuccess(ctx context.Context, ev *telemetryv1.EventEnvelope, fields map[string]interface{}) error {
	ctx, span := tracing.StartSpan(ctx, b.tracer, "handleAuthSuccess",
		tracing.EventType("authentication_success"),
		tracing.EntityID(ev.EntityId),
	)
	defer span.End()
	// Extract user
	user, _ := fields["user"].(string)
	if user == "" {
		return nil
	}

	// Add user node
	userNode := &store.Node{
		ID:          fmt.Sprintf("user:%s", user),
		Type:        store.NodeTypeUser,
		Labels:      map[string]string{},
		Criticality: 0,
	}
	if err := b.manager.AddNode(ev.TenantId, userNode); err != nil {
		return fmt.Errorf("add user node: %w", err)
	}

	// Add endpoint node (target of authentication)
	endpointNode := &store.Node{
		ID:          fmt.Sprintf("endpoint:%s", ev.EntityId),
		Type:        store.NodeTypeEndpoint,
		Labels:      map[string]string{},
		Criticality: 0,
	}
	if err := b.manager.AddNode(ev.TenantId, endpointNode); err != nil {
		return fmt.Errorf("add endpoint node: %w", err)
	}

	// Add auth edge
	authEdge := &store.Edge{
		ID:       fmt.Sprintf("auth:%s:%s:%d", user, ev.EntityId, time.Now().Unix()),
		Type:     store.EdgeTypeAuth,
		Source:   fmt.Sprintf("user:%s", user),
		Target:   fmt.Sprintf("endpoint:%s", ev.EntityId),
		Weight:   1.0,
		TTL:      1 * time.Hour, // Auth sessions expire after 1 hour
		Metadata: map[string]string{"event_type": "authentication_success"},
	}
	if err := b.manager.AddEdge(ev.TenantId, authEdge); err != nil {
		return fmt.Errorf("add auth edge: %w", err)
	}

	return nil
}

func parsePayload(payload []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// StartCleanupWorker starts a background worker that removes expired edges
func (b *Builder) StartCleanupWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Graph cleanup worker started (interval: %v)", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Graph cleanup worker stopped")
			return
		case <-ticker.C:
			results := b.manager.CleanupExpiredEdges()
			if len(results) > 0 {
				// Track cleanup metrics
				for tenantID, count := range results {
					b.metrics.GraphCleanups.WithLabelValues(tenantID).Add(float64(count))
					b.logger.Info(ctx, "Cleaned up expired edges", map[string]interface{}{
						"count": count,
					})
				}
			}

			// Update graph size metrics
			stats := b.manager.AllStats()
			for tenantID, stat := range stats {
				b.metrics.GraphNodes.WithLabelValues(tenantID).Set(float64(stat.NodeCount))
				b.metrics.GraphEdges.WithLabelValues(tenantID).Set(float64(stat.EdgeCount))
			}
		}
	}
}

// StartRiskScoringWorker starts a background worker that recomputes risk scores
func (b *Builder) StartRiskScoringWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Risk scoring worker started (interval: %v)", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Risk scoring worker stopped")
			return
		case <-ticker.C:
			// Recompute risk scores for all tenants
			stats := b.manager.AllStats()
			for tenantID := range stats {
				start := time.Now()
				graph := b.manager.GetGraph(tenantID)
				if graph != nil {
					graph.ComputeAllRiskScores()
					duration := time.Since(start).Seconds()
					b.metrics.RiskScoreTime.WithLabelValues(tenantID).Observe(duration)
					b.logger.Info(ctx, "Recomputed risk scores", map[string]interface{}{
						"node_count":    stats[tenantID].NodeCount,
						"duration_secs": duration,
					})
				}
			}
		}
	}
}

// MetricsHandler returns the Prometheus metrics HTTP handler
func (b *Builder) MetricsHandler() http.Handler {
	return b.metrics.Handler()
}
