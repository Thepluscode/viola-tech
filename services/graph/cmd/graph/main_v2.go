package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/viola/graph/internal/builder"
	"github.com/viola/graph/internal/config"
	"github.com/viola/graph/internal/store"
	sharedkafka "github.com/viola/shared/kafka"
	"github.com/viola/shared/observability/tracing"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := getenv("VIOLA_ENV", "dev")
	brokers := []string{getenv("KAFKA_BROKER", "localhost:9092")}
	metricsPort := getenv("METRICS_PORT", "9091")
	topics := sharedkafka.NewTopics(env)

	// Initialize OpenTelemetry tracing (optional)
	tracingEnabled := getenv("TRACING_ENABLED", "false") == "true"
	if tracingEnabled {
		otlpEndpoint := getenv("OTLP_ENDPOINT", "localhost:4317")
		shutdown, err := tracing.InitTracer(tracing.Config{
			ServiceName:    "graph",
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

	// Initialize graph manager
	manager := store.NewGraphManager()
	log.Println("Graph manager initialized")

	// Load crown jewels configuration
	crownJewelsPath := getenv("CROWN_JEWELS_CONFIG", "./config/crown_jewels.yaml")
	if _, err := os.Stat(crownJewelsPath); err == nil {
		crownJewels, err := config.LoadCrownJewels(crownJewelsPath)
		if err != nil {
			log.Printf("WARN: failed to load crown jewels: %v", err)
		} else {
			if err := crownJewels.ApplyToManager(manager); err != nil {
				log.Printf("WARN: failed to apply crown jewels: %v", err)
			} else {
				log.Printf("Loaded crown jewels from %s", crownJewelsPath)
			}
		}
	} else {
		log.Printf("No crown jewels config found at %s", crownJewelsPath)
	}

	// Initialize builder
	bldr := builder.New(manager)

	// Start metrics HTTP server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", bldr.MetricsHandler())
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

	// Start background workers
	go bldr.StartCleanupWorker(ctx, 5*time.Minute)
	go bldr.StartRiskScoringWorker(ctx, 5*time.Minute)

	// Kafka consumer for telemetry
	consumer, err := sharedkafka.NewConsumer(sharedkafka.ConsumerConfig{
		Brokers:     brokers,
		Topic:       topics.TelemetryNormalized,
		GroupID:     "viola." + env + ".graph.builder",
		ServiceName: "graph",
		DLQBrokers:  brokers,
		DLQTopic:    topics.DLQGraph,
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	log.Printf("Graph service consuming %s", topics.TelemetryNormalized)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down graph service...")
		cancel()

		// Shutdown metrics server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = metricsServer.Shutdown(shutdownCtx)
	}()

	// Start consuming
	if err := consumer.Run(ctx, func(ctx context.Context, msg sharedkafka.Message) error {
		return bldr.HandleTelemetryEvent(ctx, msg)
	}); err != nil {
		log.Fatalf("Consumer error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
