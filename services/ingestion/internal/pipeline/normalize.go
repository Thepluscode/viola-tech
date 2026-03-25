package pipeline

import (
  "context"
  "fmt"
  "time"

  sharedkafka "github.com/viola/shared/kafka"
  telemetryv1 "github.com/viola/shared/proto/telemetry"
)

type Normalizer struct {
  topics sharedkafka.Topics
  out *sharedkafka.Producer
}

type Config struct {
  Env string
  Brokers []string
}

func New(cfg Config) (*Normalizer, error) {
  topics := sharedkafka.NewTopics(cfg.Env)
  out, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{Brokers: cfg.Brokers, Topic: topics.TelemetryNormalized})
  if err != nil { return nil, err }
  return &Normalizer{topics: topics, out: out}, nil
}

func (n *Normalizer) Close() error { return n.out.Close() }

func (n *Normalizer) HandleRawEndpoint(ctx context.Context, msg sharedkafka.Message) error {
  codec := sharedkafka.ProtobufCodec[*telemetryv1.EventEnvelope]{Schema: "viola.telemetry.v1.EventEnvelope", New: func() *telemetryv1.EventEnvelope { return &telemetryv1.EventEnvelope{} }}
  ev, err := codec.Decode(msg.Value)
  if err != nil { return fmt.Errorf("decode envelope: %w", err) }

  if ev.TenantId == "" || msg.TenantID == "" || ev.TenantId != msg.TenantID {
    return fmt.Errorf("tenant mismatch: header=%q payload=%q", msg.TenantID, ev.TenantId)
  }

  now := time.Now().UTC()
  ev.ReceivedAt = now.Format(time.RFC3339)
  ev.EventType = canonicalEventType(ev.EventType)
  if ev.Labels == nil { ev.Labels = map[string]string{} }
  ev.Labels["stream"] = "normalized"

  b, err := codec.Encode(ev)
  if err != nil { return fmt.Errorf("encode normalized: %w", err) }

  headers := map[string]string{
    sharedkafka.HdrTenantID: ev.TenantId,
    sharedkafka.HdrRequestID: msg.RequestID,
    sharedkafka.HdrSource: "ingestion",
    sharedkafka.HdrSchema: codec.SchemaName(),
    sharedkafka.HdrEmittedAt: now.Format(time.RFC3339),
  }

  key := []byte(fmt.Sprintf("%s:%s", ev.TenantId, ev.EntityId))
  return n.out.Produce(ctx, sharedkafka.ProduceMessage{Key: key, Value: b, Headers: headers})
}

func canonicalEventType(v string) string {
  switch v {
  case "process_start", "proc_start":
    return "process_start"
  default:
    return v
  }
}
