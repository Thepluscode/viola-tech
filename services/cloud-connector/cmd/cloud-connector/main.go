package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viola/cloud-connector/internal/health"
	"github.com/viola/cloud-connector/internal/normalizer"
	"github.com/viola/cloud-connector/internal/providers/aws"
	"github.com/viola/shared/kafka"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := getenv("VIOLA_ENV", "dev")
	brokers := getenv("KAFKA_BROKER", "localhost:9092")
	tenantID := getenv("VIOLA_TENANT_ID", "t_demo")
	port := getenv("PORT", "8084")

	topics := kafka.NewTopics(env)

	// Initialize Kafka producer for cloud telemetry
	producer, err := kafka.NewProducer(kafka.ProducerConfig{
		Brokers: []string{brokers},
		Topic:   topics.TelemetryCloudRaw,
	})
	if err != nil {
		panic(fmt.Sprintf("create producer: %v", err))
	}
	defer producer.Close()

	norm := normalizer.New(producer, tenantID)

	// Start AWS CloudTrail poller
	awsRegion := getenv("AWS_REGION", "us-east-1")
	pollInterval := 60 * time.Second
	if env == "dev" {
		pollInterval = 30 * time.Second
	}

	awsPoller := aws.NewCloudTrailPoller(aws.CloudTrailConfig{
		Region:       awsRegion,
		PollInterval: pollInterval,
		LookbackMin:  15,
		TenantID:     tenantID,
	}, norm)

	go awsPoller.Run(ctx)

	// Health server
	h := health.New(awsPoller)
	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/status", h.Status)

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		fmt.Printf("cloud-connector listening on :%s\n", port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			panic(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	cancel()
	srv.Shutdown(context.Background())
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
