package testutil

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	kgo "github.com/segmentio/kafka-go"
	"github.com/viola/shared/kafka"
	telemetrypb "github.com/viola/shared/proto/telemetry"
	"google.golang.org/protobuf/proto"
)

// ---------------------------------------------------------------------------
// Kafka helpers
// ---------------------------------------------------------------------------

// KafkaBroker returns the broker address for integration tests.
// Uses KAFKA_BROKER env or defaults to localhost:9092.
func KafkaBroker() string {
	return "localhost:9092"
}

// WaitForKafka blocks until the broker is reachable or ctx expires.
func WaitForKafka(ctx context.Context, t *testing.T) {
	t.Helper()
	broker := KafkaBroker()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("kafka not reachable at %s: %v", broker, ctx.Err())
		default:
		}
		conn, err := net.DialTimeout("tcp", broker, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// ProduceRaw publishes a single protobuf-encoded EventEnvelope with proper headers.
func ProduceRaw(ctx context.Context, t *testing.T, topic string, env *telemetrypb.EventEnvelope) {
	t.Helper()
	data, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	w := &kgo.Writer{
		Addr:  kgo.TCP(KafkaBroker()),
		Topic: topic,
	}
	defer w.Close()

	headers := []kgo.Header{
		{Key: kafka.HdrTenantID, Value: []byte(env.TenantId)},
		{Key: kafka.HdrRequestID, Value: []byte("integ-" + env.EntityId)},
		{Key: kafka.HdrSource, Value: []byte(env.Source)},
		{Key: kafka.HdrSchema, Value: []byte("viola.telemetry.v1.EventEnvelope")},
		{Key: kafka.HdrEmittedAt, Value: []byte(time.Now().UTC().Format(time.RFC3339))},
	}

	if err := w.WriteMessages(ctx, kgo.Message{
		Key:     []byte(env.TenantId + ":" + env.EntityId),
		Value:   data,
		Headers: headers,
	}); err != nil {
		t.Fatalf("produce raw event: %v", err)
	}
}

// ConsumeOne reads a single message from topic within the context deadline.
func ConsumeOne(ctx context.Context, t *testing.T, topic, groupID string) kgo.Message {
	t.Helper()
	r := kgo.NewReader(kgo.ReaderConfig{
		Brokers:  []string{KafkaBroker()},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 1e6,
		MaxWait:  500 * time.Millisecond,
	})
	defer r.Close()

	msg, err := r.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("consume from %s: %v", topic, err)
	}
	return msg
}

// EnsureTopic creates a topic if it doesn't exist.
func EnsureTopic(ctx context.Context, t *testing.T, topic string, partitions int) {
	t.Helper()
	conn, err := kgo.DialContext(ctx, "tcp", KafkaBroker())
	if err != nil {
		t.Fatalf("dial kafka: %v", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		t.Fatalf("get controller: %v", err)
	}

	controllerConn, err := kgo.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		t.Fatalf("dial controller: %v", err)
	}
	defer controllerConn.Close()

	_ = controllerConn.CreateTopics(kgo.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
	})
}

// ---------------------------------------------------------------------------
// Postgres helpers
// ---------------------------------------------------------------------------

// PostgresDSN returns the connection string for integration test Postgres.
func PostgresDSN() string {
	return "postgres://postgres:postgres@localhost:5432/viola?sslmode=disable"
}

// ---------------------------------------------------------------------------
// JWT helpers
// ---------------------------------------------------------------------------

var testKey *rsa.PrivateKey

func init() {
	var err error
	testKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("generate test RSA key: " + err.Error())
	}
}

// TestRSAKey returns the shared test RSA private key.
func TestRSAKey() *rsa.PrivateKey { return testKey }

// IssueJWT creates a signed JWT for testing with the given tenant and role.
func IssueJWT(tenantID, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "viola-integration-test",
		"aud":       "viola-api",
		"sub":       "test-user@" + tenantID,
		"iat":       now.Unix(),
		"exp":       now.Add(1 * time.Hour).Unix(),
		"tenant_id": tenantID,
		"role":      role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "integ-test-key-1"
	return token.SignedString(testKey)
}

// ---------------------------------------------------------------------------
// Event builders
// ---------------------------------------------------------------------------

// NewProcessStartEnvelope builds a telemetry EventEnvelope for a process_start event.
func NewProcessStartEnvelope(tenantID, entityID string) *telemetrypb.EventEnvelope {
	payload := map[string]string{
		"pid":         "1234",
		"ppid":        "1",
		"name":        "suspicious.exe",
		"cmdline":     "suspicious.exe --exfil --target=10.0.0.1",
		"user":        "admin",
		"hash_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	payloadBytes, _ := json.Marshal(payload)

	now := time.Now().UTC().Format(time.RFC3339)
	return &telemetrypb.EventEnvelope{
		TenantId:   tenantID,
		EntityId:   entityID,
		ObservedAt: now,
		ReceivedAt: now,
		EventType:  "process_start",
		Source:      "viola-agent",
		Payload:    payloadBytes,
		Labels: map[string]string{
			"os":       "windows",
			"hostname": "workstation-42",
		},
	}
}

// NewNetworkConnectionEnvelope builds a telemetry EventEnvelope for a network_connection event.
func NewNetworkConnectionEnvelope(tenantID, entityID, destIP string) *telemetrypb.EventEnvelope {
	payload := map[string]string{
		"src_ip":   "192.168.1.100",
		"dst_ip":   destIP,
		"dst_port": "443",
		"protocol": "tcp",
		"pid":      "1234",
		"process":  "suspicious.exe",
	}
	payloadBytes, _ := json.Marshal(payload)

	now := time.Now().UTC().Format(time.RFC3339)
	return &telemetrypb.EventEnvelope{
		TenantId:   tenantID,
		EntityId:   entityID,
		ObservedAt: now,
		ReceivedAt: now,
		EventType:  "network_connection",
		Source:      "viola-agent",
		Payload:    payloadBytes,
		Labels: map[string]string{
			"os":       "windows",
			"hostname": "workstation-42",
		},
	}
}
