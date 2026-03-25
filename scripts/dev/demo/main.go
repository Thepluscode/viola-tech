// demo/main.go — Viola XDR demo data seeder
//
// Usage:
//   go run ./scripts/dev/demo/
//
// Seeds realistic security events through the full pipeline:
//   1. Produces telemetry events to Kafka (process, network, auth, cloud)
//   2. Inserts demo alerts/incidents directly into Postgres (for immediate UI)
//   3. Creates response actions
//   4. Polls gateway-api to confirm data is visible
//
// Prerequisites: docker compose up --build (all services healthy)
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	sharedkafka "github.com/viola/shared/kafka"
	telemetryv1 "github.com/viola/shared/proto/telemetry"
)

const (
	tenantID   = "tenant-dev-001"
	gatewayURL = "http://localhost:8080"
	authURL    = "http://localhost:8081"
	responseURL = "http://localhost:8083"
	kafkaBroker = "localhost:9094"
	pgDSN      = "postgres://viola:viola@localhost:5435/viola?sslmode=disable"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Println("══════════════════════════════════════════")
	fmt.Println("  Viola XDR — Demo Data Seeder")
	fmt.Println("══════════════════════════════════════════")
	fmt.Println()

	// Step 1: Seed database directly for immediate UI
	step("1. Seeding database with demo data")
	if err := seedDatabase(ctx); err != nil {
		warn("database seed: %v (continuing)", err)
	} else {
		ok("Database seeded with alerts, incidents, entities")
	}

	// Step 2: Produce telemetry events to Kafka
	step("2. Producing telemetry events to Kafka")
	if err := produceTelemetry(ctx); err != nil {
		warn("telemetry: %v (continuing)", err)
	} else {
		ok("12 telemetry events produced")
	}

	// Step 3: Create response actions
	step("3. Creating response actions")
	if err := createResponseActions(ctx); err != nil {
		warn("response actions: %v (continuing)", err)
	} else {
		ok("Response actions created")
	}

	// Step 4: Verify data in gateway-api
	step("4. Verifying data via gateway-api")
	token, err := getToken(ctx)
	if err != nil {
		warn("auth token: %v (skipping verification)", err)
	} else {
		if err := verifyGateway(ctx, token); err != nil {
			warn("gateway verification: %v", err)
		} else {
			ok("Data verified in gateway-api")
		}
	}

	fmt.Println()
	fmt.Println("══════════════════════════════════════════")
	fmt.Println("  Demo data seeded successfully!")
	fmt.Println("  Open http://localhost:3000 to view the UI")
	fmt.Println("══════════════════════════════════════════")
}

func seedDatabase(ctx context.Context) error {
	db, err := sql.Open("pgx", pgDSN)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	// Seed RBAC policies for demo tenant
	rbacPolicies := []struct{ role, resource, action string }{
		{"admin", "incidents", "read"}, {"admin", "incidents", "update"},
		{"admin", "alerts", "read"}, {"admin", "alerts", "update"},
		{"analyst", "incidents", "read"}, {"analyst", "incidents", "update"},
		{"analyst", "alerts", "read"}, {"analyst", "alerts", "update"},
		{"viewer", "incidents", "read"}, {"viewer", "alerts", "read"},
	}
	for _, p := range rbacPolicies {
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO rbac_policies (tenant_id, role, resource, action, allowed)
			VALUES ($1, $2, $3, $4, true)
			ON CONFLICT (tenant_id, role, resource, action) DO NOTHING
		`, tenantID, p.role, p.resource, p.action)
	}

	// Seed alerts
	alerts := []struct {
		id, sev, title, desc, tactic, technique string
		risk                                     float64
		conf                                     float64
		entities                                 []string
		labels                                   map[string]string
	}{
		{
			"ALT-DEMO-001", "critical",
			"LSASS Memory Access — Credential Dumping",
			"mimikatz.exe accessed lsass.exe memory on workstation-42.corp. Possible credential harvesting.",
			"Credential Access", "T1003.001", 95, 0.92,
			[]string{"host:workstation-42.corp"},
			map[string]string{"technique": "credential-dump", "target": "lsass"},
		},
		{
			"ALT-DEMO-002", "critical",
			"SMB Lateral Movement — Pass-the-Hash",
			"NTLMv2 pass-the-hash from workstation-42.corp to dc01.corp using stolen credentials.",
			"Lateral Movement", "T1550.002", 91, 0.87,
			[]string{"host:workstation-42.corp", "host:dc01.corp"},
			map[string]string{"technique": "pth", "protocol": "ntlm"},
		},
		{
			"ALT-DEMO-003", "critical",
			"Ransomware File Encryption Detected",
			"1,247 files renamed to .locked extension on fileserver-03.corp in 45 seconds.",
			"Impact", "T1486", 98, 0.96,
			[]string{"host:fileserver-03.corp"},
			map[string]string{"technique": "ransomware", "files_affected": "1247"},
		},
		{
			"ALT-DEMO-004", "high",
			"Privilege Escalation — Token Impersonation",
			"SeImpersonatePrivilege abused to obtain NT AUTHORITY\\SYSTEM on workstation-42.corp.",
			"Privilege Escalation", "T1134.001", 78, 0.74,
			[]string{"host:workstation-42.corp"},
			map[string]string{"technique": "token-impersonation"},
		},
		{
			"ALT-DEMO-005", "high",
			"DNS Tunneling — Periodic Beaconing",
			"482 DNS queries to c2.evil-domain.xyz with 61s avg interval from laptop-07.corp.",
			"Command and Control", "T1071.004", 73, 0.81,
			[]string{"host:laptop-07.corp"},
			map[string]string{"technique": "dns-tunneling", "indicator": "beaconing"},
		},
		{
			"ALT-DEMO-006", "high",
			"AWS IAM — Unauthorized AssumeRole",
			"AssumeRole to arn:aws:iam::123:role/admin from unexpected IP 198.51.100.42.",
			"Privilege Escalation", "T1078.004", 82, 0.79,
			[]string{"arn:aws:iam::123:role/admin"},
			map[string]string{"cloud_provider": "aws", "source_ip": "198.51.100.42"},
		},
		{
			"ALT-DEMO-007", "medium",
			"Defense Evasion — Security Logging Disabled",
			"CloudTrail StopLogging called on production trail from dev account.",
			"Defense Evasion", "T1562.008", 65, 0.71,
			[]string{"arn:aws:cloudtrail:us-east-1:123:trail/prod-trail"},
			map[string]string{"cloud_provider": "aws", "action": "StopLogging"},
		},
		{
			"ALT-DEMO-008", "medium",
			"Anomalous Login Pattern Detected",
			"User admin@corp.local authenticated from 3 countries in 2 hours (impossible travel).",
			"Initial Access", "T1078", 58, 0.65,
			[]string{"user:admin@corp.local"},
			map[string]string{"technique": "impossible-travel", "countries": "US,RU,CN"},
		},
		{
			"ALT-DEMO-009", "low",
			"Network Scan — Port Sweep Detected",
			"Host dev-laptop-22.corp scanned 1,024 ports on server-12.corp. Likely authorized pen test.",
			"Discovery", "T1046", 22, 0.45,
			[]string{"host:dev-laptop-22.corp", "host:server-12.corp"},
			map[string]string{"technique": "port-scan"},
		},
		{
			"ALT-DEMO-010", "low",
			"Suspicious PowerShell Encoded Command",
			"PowerShell -EncodedCommand detected on server-12.corp. Base64 decoded to benign admin script.",
			"Execution", "T1059.001", 28, 0.38,
			[]string{"host:server-12.corp"},
			map[string]string{"technique": "encoded-powershell"},
		},
	}

	for i, a := range alerts {
		labelsJSON, _ := json.Marshal(a.labels)
		created := now.Add(-time.Duration(len(alerts)-i) * 15 * time.Minute)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO alerts (tenant_id, alert_id, created_at, updated_at, status, severity,
				confidence, risk_score, title, description, mitre_tactic, mitre_technique, labels)
			VALUES ($1, $2, $3, $4, 'open', $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (tenant_id, alert_id) DO NOTHING
		`, tenantID, a.id, created, created, a.sev, a.conf, a.risk, a.title, a.desc, a.tactic, a.technique, labelsJSON)
		if err != nil {
			return fmt.Errorf("insert alert %s: %w", a.id, err)
		}

		for _, eid := range a.entities {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO alert_entities (tenant_id, alert_id, entity_id)
				VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
			`, tenantID, a.id, eid)
		}
	}
	fmt.Printf("    ✓ %d alerts seeded\n", len(alerts))

	// Seed incidents
	incidents := []struct {
		id, groupID, sev, tactic, technique, status string
		risk, conf                                   float64
		alertIDs, entities                           []string
		labels                                       map[string]string
	}{
		{
			"INC-DEMO-001", "grp-ransomware-001", "critical",
			"Lateral Movement", "T1021", "open", 98, 0.96,
			[]string{"ALT-DEMO-001", "ALT-DEMO-002", "ALT-DEMO-003", "ALT-DEMO-004"},
			[]string{"host:workstation-42.corp", "host:dc01.corp", "host:fileserver-03.corp"},
			map[string]string{"category": "ransomware", "phase": "lateral-movement"},
		},
		{
			"INC-DEMO-002", "grp-c2-001", "high",
			"Command and Control", "T1071", "ack", 73, 0.81,
			[]string{"ALT-DEMO-005"},
			[]string{"host:laptop-07.corp", "host:proxy-01.corp"},
			map[string]string{"category": "c2", "phase": "beaconing"},
		},
		{
			"INC-DEMO-003", "grp-cloud-001", "high",
			"Privilege Escalation", "T1078", "open", 82, 0.79,
			[]string{"ALT-DEMO-006", "ALT-DEMO-007"},
			[]string{"arn:aws:iam::123:role/admin", "arn:aws:cloudtrail:us-east-1:123:trail/prod-trail"},
			map[string]string{"category": "cloud-compromise", "provider": "aws"},
		},
		{
			"INC-DEMO-004", "grp-anomaly-001", "medium",
			"Initial Access", "T1078", "open", 58, 0.65,
			[]string{"ALT-DEMO-008"},
			[]string{"user:admin@corp.local"},
			map[string]string{"category": "impossible-travel"},
		},
		{
			"INC-DEMO-005", "grp-scan-001", "low",
			"Discovery", "T1046", "closed", 28, 0.45,
			[]string{"ALT-DEMO-009", "ALT-DEMO-010"},
			[]string{"host:dev-laptop-22.corp", "host:server-12.corp"},
			map[string]string{"category": "network-scan"},
		},
	}

	for i, inc := range incidents {
		labelsJSON, _ := json.Marshal(inc.labels)
		created := now.Add(-time.Duration(len(incidents)-i) * 30 * time.Minute)
		assignedTo := sql.NullString{}
		closureReason := sql.NullString{}
		if inc.status == "ack" {
			assignedTo = sql.NullString{String: "analyst@viola.corp", Valid: true}
		}
		if inc.status == "closed" {
			assignedTo = sql.NullString{String: "analyst@viola.corp", Valid: true}
			closureReason = sql.NullString{String: "Confirmed authorized pen test — false positive", Valid: true}
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO incidents (tenant_id, incident_id, correlated_group_id, created_at, updated_at,
				status, severity, max_risk_score, max_confidence,
				mitre_tactic, mitre_technique, labels, assigned_to, closure_reason,
				alert_count, hit_count)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, 0)
			ON CONFLICT (tenant_id, incident_id) DO NOTHING
		`, tenantID, inc.id, inc.groupID, created, created,
			inc.status, inc.sev, inc.risk, inc.conf,
			inc.tactic, inc.technique, labelsJSON, assignedTo, closureReason,
			len(inc.alertIDs))
		if err != nil {
			return fmt.Errorf("insert incident %s: %w", inc.id, err)
		}

		for _, eid := range inc.entities {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO incident_entities (tenant_id, incident_id, entity_id)
				VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
			`, tenantID, inc.id, eid)
		}
		for _, aid := range inc.alertIDs {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO incident_alerts (tenant_id, incident_id, alert_id)
				VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
			`, tenantID, inc.id, aid)
		}
	}
	fmt.Printf("    ✓ %d incidents seeded\n", len(incidents))

	return tx.Commit()
}

func produceTelemetry(ctx context.Context) error {
	env := getenv("VIOLA_ENV", "dev")
	topics := sharedkafka.NewTopics(env)
	prod, err := sharedkafka.NewProducer(sharedkafka.ProducerConfig{
		Brokers: []string{kafkaBroker},
		Topic:   topics.TelemetryEndpointRaw,
	})
	if err != nil {
		return fmt.Errorf("new producer: %w", err)
	}
	defer prod.Close()

	codec := sharedkafka.ProtobufCodec[*telemetryv1.EventEnvelope]{
		Schema: "viola.telemetry.v1.EventEnvelope",
		New:    func() *telemetryv1.EventEnvelope { return &telemetryv1.EventEnvelope{} },
	}

	events := []struct {
		entity, eventType string
		payload           map[string]string
	}{
		{"host:workstation-42.corp", "process_access", map[string]string{"target_process_name": "C:\\Windows\\System32\\lsass.exe", "access_mask": "0x1010", "source_process_name": "mimikatz.exe"}},
		{"host:workstation-42.corp", "process_start", map[string]string{"exe": "mimikatz.exe", "pid": "4321", "cmdline": "mimikatz.exe sekurlsa::logonpasswords"}},
		{"host:workstation-42.corp", "network_connection", map[string]string{"dst_ip": "10.0.1.5", "dst_port": "445", "protocol": "smb"}},
		{"host:dc01.corp", "auth_event", map[string]string{"type": "ntlm", "user": "admin", "result": "success", "source": "workstation-42.corp"}},
		{"host:fileserver-03.corp", "file_write", map[string]string{"path": "\\\\fileserver-03\\share\\docs\\report.docx.locked", "operation": "rename"}},
		{"host:laptop-07.corp", "dns_query", map[string]string{"domain": "c2.evil-domain.xyz", "type": "TXT", "response_size": "4096"}},
		{"host:laptop-07.corp", "network_connection", map[string]string{"dst_ip": "198.51.100.1", "dst_port": "443", "protocol": "tls"}},
		{"host:server-12.corp", "process_start", map[string]string{"exe": "powershell.exe", "cmdline": "powershell.exe -EncodedCommand SGVsbG8gV29ybGQ="}},
		{"host:proxy-01.corp", "network_connection", map[string]string{"src_ip": "10.0.2.7", "dst_ip": "198.51.100.1", "dst_port": "443"}},
		{"user:admin@corp.local", "auth_event", map[string]string{"type": "interactive", "result": "success", "source_ip": "203.0.113.1", "country": "US"}},
		{"user:admin@corp.local", "auth_event", map[string]string{"type": "interactive", "result": "success", "source_ip": "198.51.100.42", "country": "RU"}},
		{"host:dev-laptop-22.corp", "network_connection", map[string]string{"dst_ip": "10.0.1.12", "dst_port": "22", "protocol": "ssh"}},
	}

	for _, e := range events {
		payloadJSON, _ := json.Marshal(e.payload)
		now := time.Now().UTC()
		ev := &telemetryv1.EventEnvelope{
			TenantId:   tenantID,
			EntityId:   e.entity,
			ObservedAt: now.Format(time.RFC3339),
			ReceivedAt: now.Format(time.RFC3339),
			EventType:  e.eventType,
			Source:     "demo-seeder",
			Payload:    payloadJSON,
			Labels:     map[string]string{"demo": "true"},
		}

		b, err := codec.Encode(ev)
		if err != nil {
			return fmt.Errorf("encode: %w", err)
		}

		headers := map[string]string{
			sharedkafka.HdrTenantID:  tenantID,
			sharedkafka.HdrRequestID: fmt.Sprintf("demo-%d", time.Now().UnixNano()),
			sharedkafka.HdrSource:    "demo-seeder",
			sharedkafka.HdrSchema:    codec.SchemaName(),
			sharedkafka.HdrEmittedAt: now.Format(time.RFC3339),
		}

		if err := prod.Produce(ctx, sharedkafka.ProduceMessage{
			Key:     []byte(tenantID + ":" + e.entity),
			Value:   b,
			Headers: headers,
		}); err != nil {
			return fmt.Errorf("produce %s: %w", e.eventType, err)
		}
	}

	return nil
}

func createResponseActions(ctx context.Context) error {
	actions := []map[string]string{
		{"action_type": "isolate_host", "target": "workstation-42.corp", "reason": "Ransomware lateral movement detected — automatic isolation triggered", "triggered_by": "auto"},
		{"action_type": "block_ip", "target": "198.51.100.1", "reason": "C2 beaconing destination blocked at perimeter firewall", "triggered_by": "auto"},
		{"action_type": "contain_user", "target": "admin@corp.local", "reason": "Impossible travel detected — account temporarily locked", "triggered_by": "analyst@viola.corp"},
	}

	client := &http.Client{Timeout: 5 * time.Second}
	for _, a := range actions {
		body, _ := json.Marshal(a)
		resp, err := client.Post(responseURL+"/actions", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create action: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("create action got %d", resp.StatusCode)
		}
	}
	return nil
}

func getToken(ctx context.Context) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"sub":   "demo-analyst",
		"tid":   tenantID,
		"email": "analyst@viola.corp",
		"role":  "admin",
	})

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(authURL+"/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

func verifyGateway(ctx context.Context, token string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	for _, path := range []string{"/api/v1/incidents?limit=5", "/api/v1/alerts?limit=5"} {
		req, _ := http.NewRequestWithContext(ctx, "GET", gatewayURL+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("GET %s: %w", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("GET %s returned %d: %s", path, resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		json.Unmarshal(body, &result)
		if count, ok := result["count"].(float64); ok {
			fmt.Printf("    ✓ %s: %d items\n", path, int(count))
		}
	}
	return nil
}

func step(msg string) { fmt.Printf("\n▶ %s\n", msg) }
func ok(format string, args ...any) {
	fmt.Printf("  ✓ "+format+"\n", args...)
}
func warn(format string, args ...any) {
	fmt.Printf("  ⚠ "+format+"\n", args...)
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
