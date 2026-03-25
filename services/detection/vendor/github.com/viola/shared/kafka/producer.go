package kafka

import (
  "context"
  "errors"
  "time"

  kgo "github.com/segmentio/kafka-go"
)

type Producer struct {
  w       *kgo.Writer
  backoff Backoff
  retries int
}

type ProducerConfig struct {
  Brokers []string
  Topic   string
  RequiredAcks kgo.RequiredAcks
  BatchTimeout time.Duration
  Retries int
  Backoff Backoff
}

func NewProducer(cfg ProducerConfig) (*Producer, error) {
  if len(cfg.Brokers) == 0 { return nil, errors.New("producer: brokers required") }
  if cfg.Topic == "" { return nil, errors.New("producer: topic required") }
  if cfg.Retries <= 0 { cfg.Retries = 3 }
  if cfg.Backoff.Base == 0 { cfg.Backoff = Backoff{Base: 200*time.Millisecond, Max: 3*time.Second} }
  if cfg.BatchTimeout == 0 { cfg.BatchTimeout = 50*time.Millisecond }
  if cfg.RequiredAcks == 0 { cfg.RequiredAcks = kgo.RequireAll }

  w := &kgo.Writer{
    Addr: kgo.TCP(cfg.Brokers...),
    Topic: cfg.Topic,
    RequiredAcks: cfg.RequiredAcks,
    BatchTimeout: cfg.BatchTimeout,
    Async: false,
  }

  return &Producer{w: w, backoff: cfg.Backoff, retries: cfg.Retries}, nil
}

func (p *Producer) Close() error { return p.w.Close() }

type ProduceMessage struct {
  Key []byte
  Value []byte
  Headers map[string]string
}

func (p *Producer) Produce(ctx context.Context, m ProduceMessage) error {
  if err := ValidateHeaders(m.Headers); err != nil { return err }
  kHeaders := make([]kgo.Header, 0, len(m.Headers))
  for k, v := range m.Headers {
    kHeaders = append(kHeaders, kgo.Header{Key: k, Value: []byte(v)})
  }
  msg := kgo.Message{Key: m.Key, Value: m.Value, Headers: kHeaders, Time: time.Now()}

  var last error
  for attempt := 0; attempt <= p.retries; attempt++ {
    if err := p.w.WriteMessages(ctx, msg); err == nil { return nil } else { last = err }
    if attempt < p.retries {
      if err := p.backoff.Sleep(ctx, attempt); err != nil { return err }
    }
  }
  return last
}
