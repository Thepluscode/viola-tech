package kafka

import (
  "context"
  "errors"
  "runtime"
  "sync"
  "time"

  kgo "github.com/segmentio/kafka-go"
)

type Message struct {
  Topic string
  Partition int
  Offset int64
  Key []byte
  Value []byte
  Headers map[string]string

  TenantID string
  RequestID string
  Source string
  Schema string
}

type Handler func(ctx context.Context, msg Message) error

type Consumer struct {
  r *kgo.Reader
  group string
  serviceName string
  dlq *DLQPublisher
}

type ConsumerConfig struct {
  Brokers []string
  Topic string
  GroupID string
  ServiceName string

  DLQBrokers []string
  DLQTopic string

  MinBytes int
  MaxBytes int
  MaxWait time.Duration
}

func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
  if len(cfg.Brokers) == 0 { return nil, errors.New("consumer: brokers required") }
  if cfg.Topic == "" { return nil, errors.New("consumer: topic required") }
  if cfg.GroupID == "" { return nil, errors.New("consumer: group id required") }
  if cfg.ServiceName == "" { return nil, errors.New("consumer: service name required") }
  if cfg.MinBytes == 0 { cfg.MinBytes = 1e3 }
  if cfg.MaxBytes == 0 { cfg.MaxBytes = 10e6 }
  if cfg.MaxWait == 0 { cfg.MaxWait = 500 * time.Millisecond }

  r := kgo.NewReader(kgo.ReaderConfig{Brokers: cfg.Brokers, Topic: cfg.Topic, GroupID: cfg.GroupID, MinBytes: cfg.MinBytes, MaxBytes: cfg.MaxBytes, MaxWait: cfg.MaxWait})

  var dlq *DLQPublisher
  var err error
  if len(cfg.DLQBrokers) > 0 && cfg.DLQTopic != "" {
    dlq, err = NewDLQPublisher(DLQPublisherConfig{Brokers: cfg.DLQBrokers, Topic: cfg.DLQTopic, Service: cfg.ServiceName})
    if err != nil { return nil, err }
  }

  return &Consumer{r: r, group: cfg.GroupID, serviceName: cfg.ServiceName, dlq: dlq}, nil
}

func (c *Consumer) Close() error {
  if c.dlq != nil { _ = c.dlq.Close() }
  return c.r.Close()
}

func (c *Consumer) Run(ctx context.Context, h Handler) error {
  for {
    m, err := c.r.FetchMessage(ctx)
    if err != nil { return err }

    hdrs := map[string]string{}
    for _, kv := range m.Headers { hdrs[kv.Key] = string(kv.Value) }

    msg := Message{
      Topic: m.Topic, Partition: m.Partition, Offset: m.Offset,
      Key: m.Key, Value: m.Value, Headers: hdrs,
      TenantID: hdrs[HdrTenantID], RequestID: hdrs[HdrRequestID], Source: hdrs[HdrSource], Schema: hdrs[HdrSchema],
    }

    if err := ValidateHeaders(hdrs); err != nil {
      _ = c.publishToDLQ(ctx, msg, "INVALID_HEADERS", err.Error(), false)
      _ = c.r.CommitMessages(ctx, m)
      continue
    }

    if err := h(ctx, msg); err != nil {
      _ = c.publishToDLQ(ctx, msg, "HANDLER_ERROR", err.Error(), false)
      _ = c.r.CommitMessages(ctx, m)
      continue
    }

    if err := c.r.CommitMessages(ctx, m); err != nil { return err }
  }
}

// RunParallel runs the consumer with a bounded worker pool of size workers.
// Each Kafka message is dispatched to a free goroutine; the pool prevents
// unbounded goroutine creation under high-throughput bursts.
//
// Ordering guarantee: messages from the same Kafka partition key (tenant:entity)
// are processed in arrival order within their consumer-group assignment.
// Cross-partition ordering is NOT preserved — use Run() when strict ordering
// across all partitions is required.
//
// workers defaults to GOMAXPROCS when <= 0.
func (c *Consumer) RunParallel(ctx context.Context, workers int, h Handler) error {
  if workers <= 0 {
    workers = runtime.GOMAXPROCS(0)
  }

  sem := make(chan struct{}, workers)
  var wg sync.WaitGroup
  errc := make(chan error, 1)

  for {
    m, err := c.r.FetchMessage(ctx)
    if err != nil {
      wg.Wait()
      return err
    }

    hdrs := map[string]string{}
    for _, kv := range m.Headers { hdrs[kv.Key] = string(kv.Value) }

    msg := Message{
      Topic: m.Topic, Partition: m.Partition, Offset: m.Offset,
      Key: m.Key, Value: m.Value, Headers: hdrs,
      TenantID: hdrs[HdrTenantID], RequestID: hdrs[HdrRequestID], Source: hdrs[HdrSource], Schema: hdrs[HdrSchema],
    }

    if err := ValidateHeaders(hdrs); err != nil {
      _ = c.publishToDLQ(ctx, msg, "INVALID_HEADERS", err.Error(), false)
      _ = c.r.CommitMessages(ctx, m)
      continue
    }

    // Check if a worker error has been signalled.
    select {
    case workerErr := <-errc:
      wg.Wait()
      return workerErr
    default:
    }

    // Acquire a slot in the pool (blocks when all workers are busy).
    sem <- struct{}{}
    wg.Add(1)
    go func(kafkaMsg kgo.Message, appMsg Message) {
      defer func() { <-sem; wg.Done() }()
      if err := h(ctx, appMsg); err != nil {
        _ = c.publishToDLQ(ctx, appMsg, "HANDLER_ERROR", err.Error(), false)
      }
      _ = c.r.CommitMessages(ctx, kafkaMsg)
    }(m, msg)
  }
}

func (c *Consumer) publishToDLQ(ctx context.Context, msg Message, code, message string, retryable bool) error {
  if c.dlq == nil { return nil }
  snippet := msg.Value
  if len(snippet) > 4096 { snippet = snippet[:4096] }

  return c.dlq.Publish(ctx, DLQInput{
    TenantID: msg.TenantID,
    RequestID: msg.RequestID,
    Source: msg.Source,
    Schema: msg.Schema,
    OriginalTopic: msg.Topic,
    OriginalKey: string(msg.Key),
    OrigPartition: msg.Partition,
    OrigOffset: msg.Offset,
    ConsumerGroup: c.group,
    ErrorCode: code,
    ErrorMessage: message,
    Retryable: retryable,
    PayloadSnippet: snippet,
    Headers: safeHeaderSubset(msg.Headers),
  })
}

func safeHeaderSubset(h map[string]string) map[string]string {
  out := map[string]string{}
  for _, k := range []string{HdrTenantID, HdrRequestID, HdrSource, HdrSchema, HdrEmittedAt} {
    if v := h[k]; v != "" { out[k] = v }
  }
  return out
}
