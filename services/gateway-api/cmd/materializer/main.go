package main

import (
	"context"
	"log"
	"os"

	"github.com/viola/gateway-api/internal/db"
	"github.com/viola/gateway-api/internal/materializer"
	"github.com/viola/gateway-api/internal/store"
	sharedkafka "github.com/viola/shared/kafka"
)

func main() {
	ctx := context.Background()
	env := getenv("VIOLA_ENV", "dev")
	brokers := []string{getenv("KAFKA_BROKER", "localhost:9092")}
	topics := sharedkafka.NewTopics(env)

	// Database connection
	database, err := db.New(ctx, db.Config{
		Host:     getenv("PG_HOST", "localhost"),
		Port:     getenvInt("PG_PORT", 5432),
		User:     getenv("PG_USER", "postgres"),
		Password: getenv("PG_PASSWORD", "postgres"),
		Database: getenv("PG_DATABASE", "viola_gateway"),
		SSLMode:  getenv("PG_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Database connected")

	// Initialize stores
	incidentStore := store.NewIncidentStore(database.Pool())
	alertStore := store.NewAlertStore(database.Pool())

	// Initialize materializer
	mat := materializer.New(incidentStore, alertStore)

	// Consumer for alerts.created
	consumerAlertCreated, err := sharedkafka.NewConsumer(sharedkafka.ConsumerConfig{
		Brokers:     brokers,
		Topic:       topics.AlertCreated,
		GroupID:     "viola." + env + ".gateway-api.materializer.alerts-created",
		ServiceName: "gateway-api-materializer",
		DLQBrokers:  brokers,
		DLQTopic:    topics.DLQWorkers,
	})
	if err != nil {
		log.Fatalf("Failed to create alert-created consumer: %v", err)
	}
	defer consumerAlertCreated.Close()

	// Consumer for alerts.updated
	consumerAlertUpdated, err := sharedkafka.NewConsumer(sharedkafka.ConsumerConfig{
		Brokers:     brokers,
		Topic:       topics.AlertUpdated,
		GroupID:     "viola." + env + ".gateway-api.materializer.alerts-updated",
		ServiceName: "gateway-api-materializer",
		DLQBrokers:  brokers,
		DLQTopic:    topics.DLQWorkers,
	})
	if err != nil {
		log.Fatalf("Failed to create alert-updated consumer: %v", err)
	}
	defer consumerAlertUpdated.Close()

	// Consumer for incidents.upserted
	consumerIncident, err := sharedkafka.NewConsumer(sharedkafka.ConsumerConfig{
		Brokers:     brokers,
		Topic:       topics.IncidentUpserted,
		GroupID:     "viola." + env + ".gateway-api.materializer.incidents",
		ServiceName: "gateway-api-materializer",
		DLQBrokers:  brokers,
		DLQTopic:    topics.DLQWorkers,
	})
	if err != nil {
		log.Fatalf("Failed to create incident consumer: %v", err)
	}
	defer consumerIncident.Close()

	log.Printf("materializer consuming:\n  - %s\n  - %s\n  - %s",
		topics.AlertCreated, topics.AlertUpdated, topics.IncidentUpserted)

	// Run all consumers concurrently
	errChan := make(chan error, 3)

	// Alert created consumer
	go func() {
		err := consumerAlertCreated.Run(ctx, func(ctx context.Context, msg sharedkafka.Message) error {
			return mat.HandleAlertCreated(ctx, msg.Value)
		})
		errChan <- err
	}()

	// Alert updated consumer
	go func() {
		err := consumerAlertUpdated.Run(ctx, func(ctx context.Context, msg sharedkafka.Message) error {
			return mat.HandleAlertUpdated(ctx, msg.Value)
		})
		errChan <- err
	}()

	// Incident consumer
	go func() {
		err := consumerIncident.Run(ctx, func(ctx context.Context, msg sharedkafka.Message) error {
			return mat.HandleIncidentUpserted(ctx, msg.Value)
		})
		errChan <- err
	}()

	// Wait for any consumer to fail
	if err := <-errChan; err != nil {
		log.Fatalf("Consumer error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		for _, c := range v {
			if c < '0' || c > '9' {
				return fallback
			}
			i = i*10 + int(c-'0')
		}
		return i
	}
	return fallback
}
