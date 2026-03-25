# Viola Detection Rule Schema

## Overview

Viola detection rules are YAML files that define patterns to match against normalized telemetry events.

**Design goals:**
- Simple (SOC analysts can write rules without coding)
- Fast (evaluated at 50k+ events/sec)
- Expressive (support AND/OR/NOT logic, time windows, thresholds)

---

## Rule Structure

```yaml
id: viola:credential_dump_lsass
name: Credential Dumping via LSASS Access
version: 1.0.0
severity: critical           # low|med|high|critical
confidence: 0.95             # 0.0 to 1.0
category: credential_access
mitre:
  tactic: credential-access
  technique: T1003.001

description: |
  Detects direct access to the LSASS process memory, commonly used by
  credential dumping tools like Mimikatz, ProcDump, and Cobalt Strike.

event_type: process_access    # Which telemetry event type to match

# Pattern matching (all conditions must match)
match:
  - field: target_process_name
    operator: endswith
    value: lsass.exe

  - field: access_mask
    operator: contains_any
    values:
      - "0x1010"
      - "0x1410"
      - "0x1438"

  - field: source_process_name
    operator: not_in
    values:
      - wmiprvse.exe
      - taskmgr.exe
      - procexp.exe

# Optional: threshold-based detection
threshold:
  count: 1
  window: 60s          # Time window
  group_by: entity_id  # Group events by entity

# Optional: false positive suppressions
suppress_if:
  - field: source_process_path
    operator: startswith
    value: C:\Program Files\Microsoft

tags:
  - windows
  - credential-theft
  - t1003

metadata:
  author: Viola Security Team
  date: 2026-02-14
  references:
    - https://attack.mitre.org/techniques/T1003/001/
```

---

## Field Reference

### Top-Level Fields

| Field        | Type   | Required | Description                                      |
|--------------|--------|----------|--------------------------------------------------|
| id           | string | ✅        | Unique rule identifier (e.g., `viola:rule_name`) |
| name         | string | ✅        | Human-readable rule name                         |
| version      | string | ✅        | Semantic version                                 |
| severity     | enum   | ✅        | low \| med \| high \| critical                   |
| confidence   | float  | ✅        | Detection confidence (0.0 to 1.0)                |
| category     | string | ✅        | MITRE ATT&CK category                            |
| event_type   | string | ✅        | Telemetry event type to match                    |
| description  | string | ❌        | Detailed explanation                             |
| mitre        | object | ❌        | MITRE ATT&CK mapping                             |
| match        | array  | ✅        | Pattern conditions                               |
| threshold    | object | ❌        | Count-based threshold                            |
| suppress_if  | array  | ❌        | False positive suppression                       |
| tags         | array  | ❌        | Searchable tags                                  |
| metadata     | object | ❌        | Author, date, references                         |

---

## Match Conditions

### Operators

| Operator     | Description                            | Example                                      |
|--------------|----------------------------------------|----------------------------------------------|
| equals       | Exact match (case-sensitive)           | `field: name, operator: equals, value: cmd.exe` |
| equals_any   | Match any value in list                | `field: name, operator: equals_any, values: [...]` |
| contains     | String contains substring              | `field: cmdline, operator: contains, value: -enc` |
| contains_any | Contains any of the values             | `field: cmdline, operator: contains_any, values: [...]` |
| startswith   | String starts with                     | `field: path, operator: startswith, value: C:\Windows` |
| endswith     | String ends with                       | `field: name, operator: endswith, value: .exe` |
| regex        | Regular expression match               | `field: cmdline, operator: regex, value: ^powershell.*-enc` |
| not_equals   | Does not equal                         | `field: user, operator: not_equals, value: SYSTEM` |
| not_in       | Not in list of values                  | `field: name, operator: not_in, values: [...]` |
| greater_than | Numeric comparison                     | `field: port, operator: greater_than, value: 1024` |
| less_than    | Numeric comparison                     | `field: count, operator: less_than, value: 10` |

---

## Threshold Detection

```yaml
threshold:
  count: 5              # Minimum number of matches
  window: 300s          # Time window (supports s, m, h)
  group_by: entity_id   # Group by entity_id, user_id, etc.
```

**Example:** Detect 5 failed logins in 5 minutes:

```yaml
id: viola:failed_login_brute_force
event_type: authentication_failed
match:
  - field: status
    operator: equals
    value: failed
threshold:
  count: 5
  window: 300s
  group_by: user_id
```

---

## Event Type Reference

| Event Type              | Description                  | Common Fields                          |
|-------------------------|------------------------------|----------------------------------------|
| process_start           | Process creation             | process_name, cmdline, parent_process  |
| process_access          | Inter-process access         | source_process, target_process, mask   |
| file_create             | File created                 | file_path, process_name                |
| file_delete             | File deleted                 | file_path, process_name                |
| registry_set            | Registry key modified        | key_path, value_name, value_data       |
| network_connect         | Outbound connection          | dest_ip, dest_port, process_name       |
| authentication_success  | Successful authentication    | user, source_ip, method                |
| authentication_failed   | Failed authentication        | user, source_ip, reason                |
| privilege_escalation    | Privilege change             | user, new_privilege, method            |

---

## Best Practices

### 1. Avoid Overly Broad Rules

❌ **Bad:**
```yaml
match:
  - field: process_name
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
      - "IEX"
      - "DownloadString"
```

### 2. Use Suppress Conditions

```yaml
suppress_if:
  - field: process_path
    operator: startswith
    value: C:\Program Files\
```

### 3. Set Realistic Confidence Scores

- 0.95+ = High confidence, very specific pattern
- 0.80-0.94 = Medium-high confidence
- 0.60-0.79 = Medium confidence (requires analyst review)
- <0.60 = Low confidence (informational only)

### 4. Tag for Searchability

```yaml
tags:
  - windows
  - credential-theft
  - t1003
  - mimikatz
```

---

## Validation

Rules are validated on load:
- All required fields present
- `severity` is valid enum
- `confidence` is 0.0 to 1.0
- `event_type` is known
- No syntax errors in regex patterns

Invalid rules are logged but **not loaded** (engine continues with valid rules).

---

## Performance Notes

- **String matching** is fastest (use when possible)
- **Regex** is slower (use sparingly)
- **Threshold detection** requires state (memory overhead per tenant/group)
- Aim for <1ms per rule evaluation

---

## Future Enhancements

- Cross-event correlation (e.g., "process A started, then file B created")
- ML-based anomaly scoring
- Dynamic threshold adjustment per tenant
- Rule testing framework
