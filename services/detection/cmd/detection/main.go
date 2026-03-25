package main

import (
  "context"
  "log"
  "net/http"
  "os"
  "os/signal"
  "syscall"
  "time"

  "github.com/viola/detection/internal/engine"
  sharedkafka "github.com/viola/shared/kafka"
  "github.com/viola/shared/observability/tracing"
)

type Engine interface {
  HandleNormalized(ctx context.Context, msg sharedkafka.Message) error
  Close() error
  MetricsHandler() http.Handler
}

func main() {
  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  env := getenv("VIOLA_ENV", "dev")
  brokers := []string{getenv("KAFKA_BROKER", "localhost:9092")}
  rulesDir := getenv("RULES_DIR", "./rules")
  metricsPort := getenv("METRICS_PORT", "9090")
  topics := sharedkafka.NewTopics(env)

  // Initialize OpenTelemetry tracing (optional)
  tracingEnabled := getenv("TRACING_ENABLED", "false") == "true"
  if tracingEnabled {
    otlpEndpoint := getenv("OTLP_ENDPOINT", "localhost:4317")
    shutdown, err := tracing.InitTracer(tracing.Config{
      ServiceName:    "detection",
      ServiceVersion: "1.0.0",
      Environment:    env,
      OTLPEndpoint:   otlpEndpoint,
      Enabled:        true,
    })
    if err != nil {
      log.Printf("WARN: failed to initialize tracing: %v", err)
    } else {
      defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = shutdown(ctx)
      }()
      log.Printf("Tracing enabled, exporting to %s", otlpEndpoint)
    }
  }

  // Use new rule-based engine
  var eng Engine
  var err error

  log.Printf("Loading detection engine with rules from %s", rulesDir)
  eng, err = engine.NewV2(engine.ConfigV2{
    Env:      env,
    Brokers:  brokers,
    RulesDir: rulesDir,
  })
  if err != nil {
    log.Fatal(err)
  }
  defer eng.Close()

  // Start metrics HTTP server
  metricsMux := http.NewServeMux()
  metricsMux.Handle("/metrics", eng.MetricsHandler())
  metricsMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
  })

  metricsServer := &http.Server{
    Addr:    ":" + metricsPort,
    Handler: metricsMux,
  }

  go func() {
    log.Printf("Metrics server listening on :%s", metricsPort)
    if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
      log.Fatalf("Metrics server error: %v", err)
    }
  }()

  // Kafka consumer
  c, err := sharedkafka.NewConsumer(sharedkafka.ConsumerConfig{
    Brokers:     brokers,
    Topic:       topics.TelemetryNormalized,
    GroupID:     "viola." + env + ".detection.scorer",
    ServiceName: "detection",
    DLQBrokers:  brokers,
    DLQTopic:    topics.DLQDetection,
  })
  if err != nil {
    log.Fatal(err)
  }
  defer c.Close()

  // Graceful shutdown
  sigChan := make(chan os.Signal, 1)
  signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

  go func() {
    <-sigChan
    log.Println("Shutting down detection service...")
    cancel()

    // Shutdown metrics server
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()
    _ = metricsServer.Shutdown(shutdownCtx)
  }()

  log.Printf("detection consuming %s -> producing %s + %s", topics.TelemetryNormalized, topics.DetectionHit, topics.AlertCreated)
  if err := c.Run(ctx, func(ctx context.Context, msg sharedkafka.Message) error {
    return eng.HandleNormalized(ctx, msg)
  }); err != nil {
    log.Fatal(err)
  }
}

func getenv(k, d string) string {
  if v := os.Getenv(k); v != "" {
    return v
  }
  return d
}
