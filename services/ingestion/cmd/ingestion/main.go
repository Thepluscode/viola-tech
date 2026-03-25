package main

import (
  "context"
  "log"
  "os"

  "github.com/viola/ingestion/internal/pipeline"
  sharedkafka "github.com/viola/shared/kafka"
)

func main() {
  ctx := context.Background()
  env := getenv("VIOLA_ENV", "dev")
  brokers := []string{getenv("KAFKA_BROKER", "localhost:9092")}
  topics := sharedkafka.NewTopics(env)

  norm, err := pipeline.New(pipeline.Config{Env: env, Brokers: brokers})
  if err != nil { log.Fatal(err) }
  defer norm.Close()

  c, err := sharedkafka.NewConsumer(sharedkafka.ConsumerConfig{
    Brokers: brokers,
    Topic: topics.TelemetryEndpointRaw,
    GroupID: "viola."+env+".ingestion.normalize",
    ServiceName: "ingestion",
    DLQBrokers: brokers,
    DLQTopic: topics.DLQIngestion,
  })
  if err != nil { log.Fatal(err) }
  defer c.Close()

  log.Printf("ingestion consuming %s -> producing %s", topics.TelemetryEndpointRaw, topics.TelemetryNormalized)
  if err := c.Run(ctx, func(ctx context.Context, msg sharedkafka.Message) error {
    return norm.HandleRawEndpoint(ctx, msg)
  }); err != nil { log.Fatal(err) }
}

func getenv(k,d string) string { if v:=os.Getenv(k); v!="" {return v}; return d }
