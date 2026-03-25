// smoke/main.go — Viola XDR end-to-end smoke test
//
// Usage:
//   go run ./scripts/dev/smoke/
//
// What it does:
//   1. Produces one raw telemetry event to Kafka (process_access on lsass.exe)
//   2. Waits for the ingestion → detection → workers pipeline to process it
//   3. Polls the gateway-api REST API until an alert and incident appear
//   4. Prints a PASS/FAIL summary
//
// Prerequisites: docker compose up --build (all services healthy)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	sharedkafka "github.com/viola/shared/kafka"
	telemetryv1 "github.com/viola/shared/proto/telemetry"
	"github.com/viola/shared/id"
)

const (
	tenantID   = "tenant-dev-001"
	entityID   = "endpoint-win-001"
	gatewayURL = "http://localhost:8080"
	kafkaBroker = "localhost:9094" // external listener
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	requestID := id.New()
	fmt.Println("══════════════════════════════════════════")
	fmt.Println("  Viola XDR Smoke Test")
	fmt.Println("══════════════════════════════════════════")
	fmt.Printf("  Tenant:    %s\n", tenantID)
	fmt.Printf("  Entity:    %s\n", entityID)
	fmt.Printf("  RequestID: %s\n", requestID)
	fmt.Println()

	// Step 1 — produce a raw telemetry event that triggers rule 001_lsass_access
	step("1. Producing raw telemetry event to Kafka")
	if err := produceEvent(ctx, requestID); err != nil {
		fatal("produce event: %v", err)
	}
	ok("Event produced")

	// Step 2 — wait for the gateway-api to serve at least one alert
	step("2. Polling gateway-api for alert (up to 60s)")
	alertID, err := pollForAlert(ctx)
	if err != nil {
		fatal("no alert appeared: %v", err)
	}
	ok("Alert found: %s", alertID)

	// Step 3 — wait for the gateway-api to serve at least one incident
	step("3. Polling gateway-api for incident (up to 60s)")
	incidentID, err := pollForIncident(ctx)
	if err != nil {
		fatal("no incident appeared: %v", err)
	}
	ok("Incident found: %s", incidentID)

	fmt.Println()
	fmt.Println("══════════════════════════════════════════")
	fmt.Println("  RESULT: PASS ✓")
	fmt.Println("  Full pipeline: raw → ingestion → detection → workers → gateway-api")
	fmt.Println("══════════════════════════════════════════")
}

func produceEvent(ctx context.Context, requestID string) error {
	env := getenv("VIOLA_ENV", "dev")
	broker := getenv("KAFKA_BROKER_EXTERNAL", kafkaBroker)
	topics := sharedkafka.NewTopics(env)

	prod, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{
		Brokers: []string{broker},
		Topic:   topics.TelemetryEndpointRaw,
	})
	if err != nil {
		return fmt.Errorf("new producer: %w", err)
	}
	defer prod.Close()

	// Payload matches rule 001_lsass_access conditions:
	//   event_type: process_access
	//   target_process_name endswith lsass.exe
	//   access_mask contains 0x1010
	//   source_process_name not in [wmiprvse.exe, taskmgr.exe, ...]
	payload, _ := json.Marshal(map[string]string{
		"target_process_name": "C:\\Windows\\System32\\lsass.exe",
		"access_mask":         "0x1010",
		"source_process_name": "mimikatz.exe",
		"source_process_path": "C:\\Temp\\mimikatz.exe",
		"pid":                 "4321",
	})

	ev := &telemetryv1.EventEnvelope{
		TenantId:   tenantID,
		EntityId:   entityID,
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
		EventType:  "process_access",
		Source:     "smoke-test",
		Payload:    payload,
		Labels:     map[string]string{"test": "smoke"},
	}

	codec := sharedkafka.ProtobufCodec[*telemetryv1.EventEnvelope]{
		Schema: "viola.telemetry.v1.EventEnvelope",
		New:    func() *telemetryv1.EventEnvelope { return &telemetryv1.EventEnvelope{} },
	}
	b, err := codec.Encode(ev)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	headers := map[string]string{
		sharedkafka.HdrTenantID:  tenantID,
		sharedkafka.HdrRequestID: requestID,
		sharedkafka.HdrSource:    "smoke-test",
		sharedkafka.HdrSchema:    codec.SchemaName(),
		sharedkafka.HdrEmittedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return prod.Produce(ctx, sharedkafka.ProduceMessage{
		Key:     []byte(tenantID + ":" + entityID),
		Value:   b,
		Headers: headers,
	})
}

func pollForAlert(ctx context.Context) (string, error) {
	return pollResource(ctx, "/api/v1/alerts", "alerts", "alert_id")
}

func pollForIncident(ctx context.Context) (string, error) {
	return pollResource(ctx, "/api/v1/incidents", "incidents", "incident_id")
}

func pollResource(ctx context.Context, path, listKey, idKey string) (string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	url := gatewayURL + path + "?limit=1"
	client := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for %s", listKey)
		case <-ticker.C:
			resp, err := client.Get(url)
			if err != nil {
				fmt.Printf("    ↻ gateway-api not ready: %v\n", err)
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				continue
			}

			items, _ := result[listKey].([]interface{})
			if len(items) == 0 {
				fmt.Printf("    ↻ waiting for %s...\n", listKey)
				continue
			}

			first, _ := items[0].(map[string]interface{})
			resourceID, _ := first[idKey].(string)
			return resourceID, nil
		}
	}
}

func step(msg string) { fmt.Printf("\n▶ %s\n", msg) }
func ok(format string, args ...any) {
	fmt.Printf("  ✓ "+format+"\n", args...)
}
func fatal(format string, args ...any) {
	fmt.Printf("\n  ✗ "+format+"\n", args...)
	fmt.Println("\n══════════════════════════════════════════")
	fmt.Println("  RESULT: FAIL")
	fmt.Println("══════════════════════════════════════════")
	os.Exit(1)
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
