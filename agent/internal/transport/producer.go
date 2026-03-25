package transport

import (
  "context"
  "fmt"
  "time"

  sharedkafka "github.com/viola/shared/kafka"
  telemetryv1 "github.com/viola/shared/proto/telemetry"
)

type Producer struct {
  env string
  source string
  topics sharedkafka.Topics
  k *sharedkafka.Producer
  tenant string
  assetID string
}

type Config struct {
  Env string
  Brokers []string
  TenantID string
  AssetID string
  Source string
}

func NewProducer(cfg Config) (*Producer, error) {
  topics := sharedkafka.NewTopics(cfg.Env)
  kp, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{Brokers: cfg.Brokers, Topic: topics.TelemetryEndpointRaw})
  if err != nil { return nil, err }
  src := cfg.Source
  if src == "" { src = "agent" }
  return &Producer{env: cfg.Env, source: src, topics: topics, k: kp, tenant: cfg.TenantID, assetID: cfg.AssetID}, nil
}

func (p *Producer) Close() error { return p.k.Close() }

func (p *Producer) SendEndpointEvent(ctx context.Context, requestID string, eventType string, payload []byte) error {
  now := time.Now().UTC()
  env := &telemetryv1.EventEnvelope{
    TenantId: p.tenant,
    EntityId: p.assetID,
    ObservedAt: now.Format(time.RFC3339),
    ReceivedAt: now.Format(time.RFC3339),
    EventType: eventType,
    Source: p.source,
    Payload: payload,
    Labels: map[string]string{"platform": "endpoint"},
  }

  codec := sharedkafka.ProtobufCodec[*telemetryv1.EventEnvelope]{
    Schema: "viola.telemetry.v1.EventEnvelope",
    New: func() *telemetryv1.EventEnvelope { return &telemetryv1.EventEnvelope{} },
  }

  b, err := codec.Encode(env)
  if err != nil { return err }

  headers := map[string]string{
    sharedkafka.HdrTenantID: p.tenant,
    sharedkafka.HdrRequestID: requestID,
    sharedkafka.HdrSource: p.source,
    sharedkafka.HdrSchema: codec.SchemaName(),
    sharedkafka.HdrEmittedAt: now.Format(time.RFC3339),
  }

  key := []byte(fmt.Sprintf("%s:%s", p.tenant, p.assetID))
  return p.k.Produce(ctx, sharedkafka.ProduceMessage{Key: key, Value: b, Headers: headers})
}
