package main

import (
  "context"
  "os"

  "github.com/viola/agent/internal/transport"
)

func main() {
  ctx := context.Background()
  env := getenv("VIOLA_ENV", "dev")
  brokers := []string{getenv("KAFKA_BROKER", "localhost:9092")}
  tenant := getenv("VIOLA_TENANT_ID", "t_demo")
  asset := getenv("VIOLA_ASSET_ID", "host-1")

  p, err := transport.NewProducer(transport.Config{Env: env, Brokers: brokers, TenantID: tenant, AssetID: asset, Source: "agent"})
  if err != nil { panic(err) }
  defer p.Close()

  payload := []byte(`{"pid":123,"exe":"cmd.exe","ppid":1}`)
  if err := p.SendEndpointEvent(ctx, "req_demo_1", "process_start", payload); err != nil { panic(err) }
}

func getenv(k, d string) string {
  if v := os.Getenv(k); v != "" { return v }
  return d
}
