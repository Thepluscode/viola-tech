package kafka

import (
  "context"
  "errors"
  "fmt"
  "time"

  kgo "github.com/segmentio/kafka-go"
  "google.golang.org/protobuf/proto"

  "github.com/viola/shared/id"
  dlqv1 "github.com/viola/shared/proto/dlq"
)

type DLQPublisher struct {
  producer *Producer
  service string
}

type DLQPublisherConfig struct {
  Brokers []string
  Topic string
  Service string
}

func NewDLQPublisher(cfg DLQPublisherConfig) (*DLQPublisher, error) {
  if cfg.Service == "" { return nil, errors.New("dlq: service required") }
  pr, err := NewProducer(ProducerConfig{Brokers: cfg.Brokers, Topic: cfg.Topic, RequiredAcks: kgo.RequireAll, Retries: 3, Backoff: Backoff{Base: 200*time.Millisecond, Max: 3*time.Second}})
  if err != nil { return nil, err }
  return &DLQPublisher{producer: pr, service: cfg.Service}, nil
}

func (d *DLQPublisher) Close() error { return d.producer.Close() }

type DLQInput struct {
  TenantID string
  RequestID string
  Source string
  Schema string
  OriginalTopic string
  OriginalKey string
  OrigPartition int
  OrigOffset int64
  ConsumerGroup string
  ErrorCode string
  ErrorMessage string
  Retryable bool
  PayloadSnippet []byte
  Headers map[string]string
}

func (d *DLQPublisher) Publish(ctx context.Context, in DLQInput) error {
  if in.TenantID == "" { return errors.New("dlq: tenant_id required") }
  if in.RequestID == "" { return errors.New("dlq: request_id required") }
  if in.ErrorCode == "" { in.ErrorCode = "UNSPECIFIED" }
  if len(in.PayloadSnippet) > 4096 { in.PayloadSnippet = in.PayloadSnippet[:4096] }

  ev := &dlqv1.DlqEvent{
    TenantId: in.TenantID,
    DlqId: id.New(),
    FailedAt: time.Now().UTC().Format(time.RFC3339),
    ProducerService: d.service,
    ConsumerGroup: in.ConsumerGroup,
    OriginalTopic: in.OriginalTopic,
    OriginalPartition: int32(in.OrigPartition),
    OriginalOffset: in.OrigOffset,
    OriginalKey: in.OriginalKey,
    Schema: in.Schema,
    RequestId: in.RequestID,
    Source: in.Source,
    ErrorCode: in.ErrorCode,
    ErrorMessage: sanitize(in.ErrorMessage),
    Retryable: in.Retryable,
    PayloadSnippet: in.PayloadSnippet,
    Headers: in.Headers,
  }

  b, err := proto.Marshal(ev)
  if err != nil { return err }

  headers := map[string]string{
    HdrTenantID: in.TenantID,
    HdrRequestID: in.RequestID,
    HdrSource: d.service,
    HdrSchema: "viola.dlq.v1.DlqEvent",
    HdrEmittedAt: time.Now().UTC().Format(time.RFC3339),
  }

  key := []byte(fmt.Sprintf("%s:%s", in.TenantID, ev.DlqId))
  return d.producer.Produce(ctx, ProduceMessage{Key: key, Value: b, Headers: headers})
}

func sanitize(s string) string {
  out := make([]rune, 0, len(s))
  for _, r := range s {
    if r == '\n' || r == '\r' { continue }
    out = append(out, r)
  }
  if len(out) > 512 { return string(out[:512]) }
  return string(out)
}
