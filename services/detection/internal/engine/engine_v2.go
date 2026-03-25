package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/viola/detection/internal/rule"
	"github.com/viola/shared/correlation"
	"github.com/viola/shared/id"
	sharedkafka "github.com/viola/shared/kafka"
	"github.com/viola/shared/observability/logging"
	"github.com/viola/shared/observability/metrics"
	"github.com/viola/shared/observability/tracing"
	securityv1 "github.com/viola/shared/proto/security"
	telemetryv1 "github.com/viola/shared/proto/telemetry"
)

// indexedRule pairs a Rule with its optional Bloom filter pre-screener.
// The filter is nil for rules with no "equals"/"equals_any" conditions.
type indexedRule struct {
	r      *rule.Rule
	bloom  *rule.RuleBloomFilter
}

type EngineV2 struct {
	// ruleIndex maps EventType → indexed rules for that type.
	// wildcardRules holds rules with EventType="" that evaluate every event.
	// Building this index at startup gives O(1) dispatch vs O(N) linear scan.
	ruleIndex     map[string][]indexedRule
	wildcardRules []indexedRule

	tracker        *rule.ThresholdTracker
	topics         sharedkafka.Topics
	hitProd        *sharedkafka.Producer
	alertProd      *sharedkafka.Producer
	partitionStrat *sharedkafka.PartitionKeyStrategy

	// Observability
	metrics *metrics.Registry
	logger  *logging.Logger
	tracer  trace.Tracer
}

type ConfigV2 struct {
	Env      string
	Brokers  []string
	RulesDir string
}

// buildRuleIndex constructs the EventType → []indexedRule dispatch map.
// Rules without an EventType go into wildcardRules.
// Each rule gets a Bloom filter for exact-match pre-screening.
func buildRuleIndex(rules []*rule.Rule) (map[string][]indexedRule, []indexedRule) {
	idx := make(map[string][]indexedRule)
	var wildcards []indexedRule
	for _, r := range rules {
		ir := indexedRule{
			r:     r,
			bloom: rule.BuildBloomFilter(r),
		}
		if r.EventType == "" {
			wildcards = append(wildcards, ir)
		} else {
			idx[r.EventType] = append(idx[r.EventType], ir)
		}
	}
	return idx, wildcards
}

func NewV2(cfg ConfigV2) (*EngineV2, error) {
	// Load rules
	rules, err := rule.LoadRules(cfg.RulesDir)
	if err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}
	log.Printf("Loaded %d detection rules", len(rules))

	// Build dispatch index — O(1) lookup per event type
	ruleIndex, wildcardRules := buildRuleIndex(rules)
	log.Printf("Rule index: %d event types, %d wildcard rules", len(ruleIndex), len(wildcardRules))

	topics := sharedkafka.NewTopics(cfg.Env)

	// Kafka producers
	hitProd, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{
		Brokers: cfg.Brokers,
		Topic:   topics.DetectionHit,
	})
	if err != nil {
		return nil, err
	}

	alertProd, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{
		Brokers: cfg.Brokers,
		Topic:   topics.AlertCreated,
	})
	if err != nil {
		_ = hitProd.Close()
		return nil, err
	}

	// Initialize observability
	metricsRegistry := metrics.NewRegistry("detection")
	logger := logging.New("detection", logging.INFO)
	tracer := tracing.Tracer("detection")

	return &EngineV2{
		ruleIndex:      ruleIndex,
		wildcardRules:  wildcardRules,
		tracker:        rule.NewThresholdTracker(),
		topics:         topics,
		hitProd:        hitProd,
		alertProd:      alertProd,
		partitionStrat: sharedkafka.NewPartitionKeyStrategy(cfg.Env),
		metrics:        metricsRegistry,
		logger:         logger,
		tracer:         tracer,
	}, nil
}

func (e *EngineV2) Close() error {
	_ = e.hitProd.Close()
	return e.alertProd.Close()
}

// MetricsHandler returns the Prometheus metrics HTTP handler
func (e *EngineV2) MetricsHandler() http.Handler {
	return e.metrics.Handler()
}

func (e *EngineV2) HandleNormalized(ctx context.Context, msg sharedkafka.Message) error {
	start := time.Now()

	// Add request context for logging
	ctx = logging.WithContext(ctx, msg.RequestID, msg.TenantID)

	// Start tracing span
	ctx, span := tracing.StartSpan(ctx, e.tracer, "HandleNormalized",
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
		e.metrics.EventsErrors.WithLabelValues(msg.TenantID, "decode_error").Inc()
		tracing.RecordError(ctx, err)
		return fmt.Errorf("decode envelope: %w", err)
	}

	if ev.TenantId == "" || ev.TenantId != msg.TenantID {
		e.metrics.EventsErrors.WithLabelValues(msg.TenantID, "tenant_mismatch").Inc()
		return fmt.Errorf("tenant mismatch")
	}

	// Track event processed
	e.metrics.EventsProcessed.WithLabelValues(ev.TenantId, ev.EventType).Inc()

	// Parse payload to fields
	fields, err := parsePayload(ev.Payload)
	if err != nil {
		e.logger.Warnf(ctx, "failed to parse payload for %s: %v", ev.EventType, err)
		fields = make(map[string]string)
	}

	// Build rule.Event
	ruleEvent := &rule.Event{
		TenantID:  ev.TenantId,
		EntityID:  ev.EntityId,
		EventType: ev.EventType,
		Fields:    fields,
	}

	// Dispatch: evaluate only rules indexed for this event type + wildcard rules.
	// This replaces the O(N) full scan with an O(K) scan where K = rules for this type.
	candidates := e.ruleIndex[ev.EventType]
	if len(e.wildcardRules) > 0 {
		if len(candidates) == 0 {
			candidates = e.wildcardRules
		} else {
			combined := make([]indexedRule, 0, len(candidates)+len(e.wildcardRules))
			combined = append(combined, candidates...)
			combined = append(combined, e.wildcardRules...)
			candidates = combined
		}
	}

	matchedRules := 0
	for _, ir := range candidates {
		// Bloom pre-screen: skip full condition evaluation when the event's
		// exact-match field values are definitely absent from the filter.
		if !rule.EventMatchesBloom(ir.bloom, ir.r, ruleEvent) {
			continue
		}

		if ir.r.Match(ruleEvent) {
			// Check threshold (if applicable)
			if !e.tracker.Check(ir.r, ruleEvent) {
				continue // Threshold not met
			}

			matchedRules++

			// Rule matched! Publish detection hit
			if err := e.publishHit(ctx, ir.r, ev, msg.RequestID); err != nil {
				e.logger.Errorf(ctx, "failed to publish hit for rule %s: %v", ir.r.ID, err)
				e.metrics.EventsErrors.WithLabelValues(ev.TenantId, "hit_publish_error").Inc()
				continue
			}

			// Publish alert
			if err := e.publishAlert(ctx, ir.r, ev, msg.RequestID); err != nil {
				e.logger.Errorf(ctx, "failed to publish alert for rule %s: %v", ir.r.ID, err)
				e.metrics.EventsErrors.WithLabelValues(ev.TenantId, "alert_publish_error").Inc()
				continue
			}

			e.logger.Info(ctx, "Detection matched", map[string]interface{}{
				"rule_id":   ir.r.ID,
				"rule_name": ir.r.Name,
				"entity_id": ev.EntityId,
				"severity":  ir.r.Severity,
			})
		}
	}

	// Track processing time
	duration := time.Since(start).Seconds()
	e.metrics.ProcessingTime.WithLabelValues(ev.TenantId, ev.EventType).Observe(duration)

	if matchedRules > 0 {
		tracing.AddEvent(ctx, "rules_matched", tracing.RuleID(fmt.Sprintf("%d", matchedRules)))
	}

	return nil
}

func (e *EngineV2) publishHit(ctx context.Context, r *rule.Rule, ev *telemetryv1.EventEnvelope, requestID string) error {
	now := time.Now().UTC()
	groupID := correlation.GroupID(ev.TenantId, r.ID, ev.EntityId, now, correlation.Bucket15m)

	hit := &securityv1.DetectionHit{
		TenantId:          ev.TenantId,
		HitId:             id.New(),
		DetectedAt:        now.Format(time.RFC3339),
		RuleId:            r.ID,
		RuleName:          r.Name,
		RuleVersion:       r.Version,
		Category:          r.Category,
		Severity:          r.Severity,
		Confidence:        r.Confidence,
		EntityIds:         []string{ev.EntityId},
		MitreTactic:       mitreTactic(r),
		MitreTechnique:    mitreTechnique(r),
		CorrelatedGroupId: groupID,
		RequestId:         requestID,
		Source:            "detection",
		Evidence: []*securityv1.Evidence{{
			ObservedAt:     ev.ObservedAt,
			EventType:      ev.EventType,
			EntityId:       ev.EntityId,
			Summary:        fmt.Sprintf("Matched rule: %s", r.Name),
			PayloadSnippet: capBytes(ev.Payload, 1024),
			Labels:         map[string]string{"rule_id": r.ID},
		}},
		Tags: r.Tags,
	}

	hitCodec := sharedkafka.ProtobufCodec[*securityv1.DetectionHit]{
		Schema: "viola.security.v1.DetectionHit",
		New:    func() *securityv1.DetectionHit { return &securityv1.DetectionHit{} },
	}
	hitBytes, err := hitCodec.Encode(hit)
	if err != nil {
		return err
	}

	hitHdr := map[string]string{
		sharedkafka.HdrTenantID:  ev.TenantId,
		sharedkafka.HdrRequestID: requestID,
		sharedkafka.HdrSource:    "detection",
		sharedkafka.HdrSchema:    hitCodec.SchemaName(),
		sharedkafka.HdrEmittedAt: now.Format(time.RFC3339),
	}

	hitKey := e.partitionStrat.KeyForDetectionHit(ev.TenantId, ev.EntityId)
	return e.hitProd.Produce(ctx, sharedkafka.ProduceMessage{
		Key:     hitKey,
		Value:   hitBytes,
		Headers: hitHdr,
	})
}

func (e *EngineV2) publishAlert(ctx context.Context, r *rule.Rule, ev *telemetryv1.EventEnvelope, requestID string) error {
	// Start span for alert publishing
	ctx, span := tracing.StartSpan(ctx, e.tracer, "publishAlert",
		tracing.RuleID(r.ID),
		tracing.EntityID(ev.EntityId),
	)
	defer span.End()

	now := time.Now().UTC()
	groupID := correlation.GroupID(ev.TenantId, r.ID, ev.EntityId, now, correlation.Bucket15m)

	alert := &securityv1.Alert{
		TenantId:          ev.TenantId,
		AlertId:           id.New(),
		CreatedAt:         now.Format(time.RFC3339),
		UpdatedAt:         now.Format(time.RFC3339),
		Status:            "open",
		Severity:          r.Severity,
		Confidence:        r.Confidence,
		RiskScore:         calculateRiskScore(r),
		Title:             r.Name,
		Description:       r.Description,
		EntityIds:         []string{ev.EntityId},
		DetectionHitIds:   []string{}, // TODO: correlate hits
		MitreTactic:       mitreTactic(r),
		MitreTechnique:    mitreTechnique(r),
		Labels:            convertTags(r.Tags),
		RequestId:         requestID,
		CorrelatedGroupId: groupID,
	}

	// Track alert metrics
	e.metrics.AlertsGenerated.WithLabelValues(ev.TenantId, r.ID, r.Severity).Inc()
	e.metrics.AlertRiskScore.WithLabelValues(ev.TenantId, r.Severity).Observe(alert.RiskScore)

	alertCodec := sharedkafka.ProtobufCodec[*securityv1.Alert]{
		Schema: "viola.security.v1.Alert",
		New:    func() *securityv1.Alert { return &securityv1.Alert{} },
	}
	alertBytes, err := alertCodec.Encode(alert)
	if err != nil {
		tracing.RecordError(ctx, err)
		return err
	}

	alertHdr := map[string]string{
		sharedkafka.HdrTenantID:  ev.TenantId,
		sharedkafka.HdrRequestID: requestID,
		sharedkafka.HdrSource:    "detection",
		sharedkafka.HdrSchema:    alertCodec.SchemaName(),
		sharedkafka.HdrEmittedAt: now.Format(time.RFC3339),
	}

	alertKey := e.partitionStrat.KeyForAlert(ev.TenantId, alert.AlertId)
	err = e.alertProd.Produce(ctx, sharedkafka.ProduceMessage{
		Key:     alertKey,
		Value:   alertBytes,
		Headers: alertHdr,
	})

	if err != nil {
		tracing.RecordError(ctx, err)
	} else {
		tracing.AddEvent(ctx, "alert_published", tracing.AlertID(alert.AlertId))
	}

	return err
}

func parsePayload(payload []byte) (map[string]string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	fields := make(map[string]string)
	flatten("", data, fields)
	return fields, nil
}

func flatten(prefix string, data map[string]interface{}, out map[string]string) {
	for k, v := range data {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			out[key] = val
		case float64:
			out[key] = fmt.Sprintf("%.0f", val)
		case bool:
			out[key] = fmt.Sprintf("%t", val)
		case map[string]interface{}:
			flatten(key, val, out)
		default:
			out[key] = fmt.Sprintf("%v", v)
		}
	}
}

func mitreTactic(r *rule.Rule) string {
	if r.MITRE != nil {
		return r.MITRE.Tactic
	}
	return ""
}

func mitreTechnique(r *rule.Rule) string {
	if r.MITRE != nil {
		return r.MITRE.Technique
	}
	return ""
}

// mitreStageWeight maps ATT&CK tactic stage names to weights reflecting attacker
// progress through the kill chain. Later stages indicate greater dwell time / impact.
var mitreStageWeight = map[string]float64{
	"reconnaissance":   0.10,
	"resource-development": 0.15,
	"initial-access":   0.30,
	"execution":        0.40,
	"persistence":      0.50,
	"privilege-escalation": 0.65,
	"defense-evasion":  0.60,
	"credential-access": 0.70,
	"discovery":        0.45,
	"lateral-movement": 0.80,
	"collection":       0.75,
	"command-and-control": 0.85,
	"exfiltration":     0.90,
	"impact":           1.00,
}

// severityWeight maps severity strings to base score components.
var severityWeight = map[string]float64{
	"low":      0.25,
	"med":      0.50,
	"high":     0.75,
	"critical": 1.00,
}

// calculateRiskScore uses a multi-factor formula:
//
//	risk = (severity_weight × 0.4 + mitre_stage_weight × 0.3 + confidence × 0.3) × 100
//
// This weighs severity (impact potential) at 40%, MITRE kill-chain position
// (attacker progress) at 30%, and model confidence at 30%.
// Result is always in [0, 100].
func calculateRiskScore(r *rule.Rule) float64 {
	sw := severityWeight[r.Severity]

	mw := 0.0
	if r.MITRE != nil {
		mw = mitreStageWeight[r.MITRE.Tactic]
	}

	score := (sw*0.4 + mw*0.3 + r.Confidence*0.3) * 100.0
	if score > 100.0 {
		score = 100.0
	}
	return score
}

func convertTags(tags []string) map[string]string {
	if len(tags) == 0 {
		return map[string]string{}
	}
	m := make(map[string]string)
	for i, tag := range tags {
		m[fmt.Sprintf("tag_%d", i)] = tag
	}
	return m
}

func capBytes(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return b[:max]
}
