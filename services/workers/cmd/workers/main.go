package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/viola/workers/internal/incident"
	sharedkafka "github.com/viola/shared/kafka"
)

func main() {
	// Root context — cancelled on SIGTERM / SIGINT for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	env := getenv("VIOLA_ENV", "dev")
	brokers := []string{getenv("KAFKA_BROKER", "localhost:9092")}
	topics := sharedkafka.NewTopics(env)

	// Connect to Postgres via pgxpool (H2: no lib/pq).
	pool, err := pgxpool.New(ctx, getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/viola?sslmode=disable"))
	if err != nil {
		log.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	agg, err := incident.New(incident.Config{Pool: pool, Env: env, Brokers: brokers})
	if err != nil {
		log.Fatalf("incident.New: %v", err)
	}
	defer agg.Close()

	c, err := sharedkafka.NewConsumer(sharedkafka.ConsumerConfig{
		Brokers:     brokers,
		Topic:       topics.AlertCreated,
		GroupID:     "viola." + env + ".workers.incident_agg",
		ServiceName: "workers",
		DLQBrokers:  brokers,
		DLQTopic:    topics.DLQWorkers,
	})
	if err != nil {
		log.Fatalf("NewConsumer: %v", err)
	}
	defer c.Close()

	// Worker pool size: WORKERS env var, else 2× GOMAXPROCS (CPU-bound I/O mix).
	workers := runtime.GOMAXPROCS(0) * 2
	if w := getenv("WORKERS", ""); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			workers = n
		}
	}

	log.Printf("workers: consuming %s -> producing %s (pool=%d)", topics.AlertCreated, topics.IncidentUpserted, workers)

	// RunParallel dispatches each Kafka message to a goroutine from a bounded pool,
	// enabling multi-partition throughput without unbounded goroutine growth.
	if err := c.RunParallel(ctx, workers, func(ctx context.Context, msg sharedkafka.Message) error {
		return agg.HandleAlertCreated(ctx, msg)
	}); err != nil {
		log.Fatalf("consumer: %v", err)
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
