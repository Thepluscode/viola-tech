# Viola Detection Rules

This directory contains YAML-based detection rules that power the Viola detection engine.

---

## Quick Start

```bash
# Test rule validation
cd services/detection
go run cmd/detection/main.go

# Should output:
# Loaded 10 detection rules
```

---

## Available Rules

| Rule ID                                  | Severity | MITRE Technique | Description                          |
|------------------------------------------|----------|-----------------|--------------------------------------|
| `viola:credential_dump_lsass`           | critical | T1003.001       | LSASS credential dumping             |
| `viola:lateral_movement_psexec`         | high     | T1570           | PSExec lateral movement              |
| `viola:exec_powershell_encoded`         | high     | T1059.001       | Encoded PowerShell commands          |
| `viola:persist_admin_account_creation`  | high     | T1136.001       | Suspicious admin account creation    |
| `viola:persist_registry_run_key`        | med      | T1547.001       | Registry run key persistence         |
| `viola:defense_evasion_event_log_clear` | critical | T1070.001       | Event log clearing                   |
| `viola:exec_rundll32_suspicious`        | high     | T1218.011       | Suspicious rundll32 usage            |
| `viola:credential_brute_force`          | high     | T1110.001       | Failed login brute force (threshold) |
| `viola:persist_scheduled_task`          | med      | T1053.005       | Scheduled task creation              |
| `viola:exec_wmi_suspicious`             | high     | T1047           | Suspicious WMI execution             |

---

## Rule Coverage

**MITRE ATT&CK Tactics:**
- Credential Access (2 rules)
- Execution (5 rules)
- Persistence (3 rules)
- Lateral Movement (2 rules)
- Defense Evasion (1 rule)

**Total Rules:** 10
**Threshold-based:** 1 (brute force)

---

## Adding New Rules

1. Create a new `.yaml` file in this directory
2. Follow the schema defined in `schema.md`
3. Restart the detection service (rules are loaded at startup)

**Example:**

```yaml
id: viola:my_custom_rule
name: My Custom Detection Rule
version: 1.0.0
severity: high
confidence: 0.90
category: execution
event_type: process_start

match:
  - field: process_name
    operator: equals
    value: malware.exe

tags:
  - custom
```

---

## Testing Rules

### Manual Testing

Create a test telemetry event:

```json
{
  "tenant_id": "test-tenant",
  "entity_id": "host-123",
  "event_type": "process_start",
  "observed_at": "2026-02-14T12:00:00Z",
  "payload": {
    "process_name": "powershell.exe",
    "cmdline": "powershell.exe -EncodedCommand ABC123",
    "parent_process_name": "explorer.exe",
    "user": "alice"
  }
}
```

**Expected match:** `viola:exec_powershell_encoded`

### Unit Testing

```go
package rule_test

import (
	"testing"
	"github.com/viola/detection/internal/rule"
)

func TestPowerShellEncodedRule(t *testing.T) {
	r, err := rule.LoadRule("003_powershell_encoded.yaml")
	if err != nil {
		t.Fatal(err)
	}

	event := &rule.Event{
		TenantID:  "test",
		EntityID:  "host-1",
		EventType: "process_start",
		Fields: map[string]string{
			"process_name": "powershell.exe",
			"cmdline":      "powershell.exe -EncodedCommand ABC123",
		},
	}

	if !r.Match(event) {
		t.Error("Expected rule to match")
	}
}
```

---

## Rule Performance

**Benchmarks** (single rule evaluation):

- String matching: ~100 ns
- Regex matching: ~500 ns
- Threshold tracking: ~200 ns

**Throughput:** ~50k EPS (events per second) per CPU core

---

## False Positive Management

### Suppression Conditions

Add `suppress_if` conditions to reduce false positives:

```yaml
suppress_if:
  - field: process_path
    operator: startswith
    value: C:\Program Files\
```

### Per-Tenant Overrides

**Future:** Support per-tenant rule customization (disable rules, adjust thresholds).

---

## Rule Versioning

Rules follow semantic versioning:
- **Major** version: Breaking changes to rule logic
- **Minor** version: Non-breaking improvements (new conditions)
- **Patch** version: Documentation or metadata updates

**Example:** `1.2.3`
- `1`: Major detection logic version
- `2`: Added suppression condition
- `3`: Updated MITRE mapping

---

## Troubleshooting

### Rule Not Loading

**Symptom:** `WARN: failed to load rule` in logs

**Causes:**
1. Invalid YAML syntax
2. Missing required fields
3. Invalid regex pattern
4. Confidence out of range (0.0 - 1.0)

**Fix:** Run validation:

```bash
go run cmd/detection/main.go
# Check logs for specific error
```

### Rule Not Matching

**Symptom:** Expected detection but no hit published

**Debug steps:**
1. Check `event_type` matches
2. Verify field names in telemetry match rule conditions
3. Check if `suppress_if` is triggering
4. Add debug logging to `matcher.go`

### High False Positive Rate

**Solutions:**
1. Lower `confidence` score
2. Add more specific match conditions
3. Add `suppress_if` conditions
4. Increase threshold count (for threshold-based rules)

---

## Best Practices

### 1. Start Specific, Then Broaden

❌ **Bad:**
```yaml
match:
  - field: cmdline
    operator: contains
    value: powershell
```

✅ **Good:**
```yaml
match:
  - field: process_name
    operator: equals
    value: powershell.exe
  - field: cmdline
    operator: contains_any
    values:
      - "-EncodedCommand"
      - "-WindowStyle Hidden"
```

### 2. Use Suppressions for Known Good

```yaml
suppress_if:
  - field: parent_process_name
    operator: equals_any
    values:
      - svchost.exe
      - services.exe
```

### 3. Tag for Search and Filtering

```yaml
tags:
  - windows
  - credential-theft
  - t1003
  - mimikatz
```

### 4. Document Edge Cases

```yaml
description: |
  Note: This rule may false positive on SysInternals tools like ProcDump
  when used by administrators for troubleshooting.
```

---

## Next Steps

1. **Add Linux rules** - Currently Windows-focused
2. **Add cloud rules** - AWS/Azure suspicious API calls
3. **Cross-event correlation** - Multi-stage attack detection
4. **ML-based anomaly detection** - Complement rule-based detection
5. **Automated rule testing** - CI/CD validation pipeline

---

## References

- [MITRE ATT&CK](https://attack.mitre.org/)
- [Sigma Rule Format](https://github.com/SigmaHQ/sigma)
- [Rule Schema Documentation](./schema.md)
