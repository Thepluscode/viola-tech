"""Feature extraction from telemetry events for anomaly detection."""

import hashlib
import json
from dataclasses import dataclass, field
from typing import Any

import numpy as np


@dataclass
class TelemetryFeatures:
    """Extracted feature vector from a telemetry event."""

    tenant_id: str
    entity_id: str
    event_type: str
    features: np.ndarray
    raw_event: dict = field(default_factory=dict)


def extract_features(tenant_id: str, entity_id: str, event_type: str,
                     payload: dict, labels: dict) -> TelemetryFeatures:
    """Extract a numeric feature vector from a telemetry event.

    Feature vector (12 dimensions):
    [0]  event_type_hash       - Hashed event type (normalized 0-1)
    [1]  payload_size          - Size of payload in bytes (log-scaled)
    [2]  field_count           - Number of fields in payload
    [3]  has_network_fields    - Binary: contains IP/port fields
    [4]  has_process_fields    - Binary: contains pid/exe fields
    [5]  has_auth_fields       - Binary: contains user/token fields
    [6]  hour_of_day           - Normalized hour (0-1)
    [7]  numeric_field_entropy - Entropy of numeric values in payload
    [8]  string_length_mean    - Mean length of string values
    [9]  label_count           - Number of labels
    [10] source_hash           - Hashed source field
    [11] cmdline_length        - Length of command line if present (log-scaled)
    """
    features = np.zeros(12, dtype=np.float64)

    # [0] Event type hash
    features[0] = _hash_to_float(event_type)

    # [1] Payload size (log-scaled)
    payload_str = json.dumps(payload)
    features[1] = np.log1p(len(payload_str))

    # [2] Field count
    features[2] = len(payload)

    # [3-5] Field type indicators
    network_keys = {"src_ip", "dst_ip", "src_port", "dst_port", "protocol", "ip", "port"}
    process_keys = {"pid", "ppid", "exe", "name", "cmdline", "process", "hash_sha256"}
    auth_keys = {"user", "username", "token", "session", "credential", "password"}

    payload_keys = set(k.lower() for k in payload.keys())
    features[3] = 1.0 if payload_keys & network_keys else 0.0
    features[4] = 1.0 if payload_keys & process_keys else 0.0
    features[5] = 1.0 if payload_keys & auth_keys else 0.0

    # [6] Hour of day (placeholder — would use observed_at in production)
    import time
    features[6] = (time.localtime().tm_hour) / 24.0

    # [7] Numeric field entropy
    numeric_vals = []
    for v in payload.values():
        try:
            numeric_vals.append(float(v))
        except (TypeError, ValueError):
            pass
    if numeric_vals:
        arr = np.array(numeric_vals)
        arr_norm = arr / (np.max(np.abs(arr)) + 1e-10)
        features[7] = _entropy(arr_norm)

    # [8] String length mean
    str_lengths = [len(str(v)) for v in payload.values() if isinstance(v, str)]
    features[8] = np.mean(str_lengths) if str_lengths else 0.0

    # [9] Label count
    features[9] = len(labels)

    # [10] Source hash
    source = labels.get("source", payload.get("source", ""))
    features[10] = _hash_to_float(str(source))

    # [11] Command line length
    cmdline = payload.get("cmdline", payload.get("command_line", ""))
    features[11] = np.log1p(len(str(cmdline)))

    return TelemetryFeatures(
        tenant_id=tenant_id,
        entity_id=entity_id,
        event_type=event_type,
        features=features,
        raw_event=payload,
    )


def _hash_to_float(s: str) -> float:
    """Deterministically hash a string to a float in [0, 1]."""
    h = hashlib.sha256(s.encode()).hexdigest()[:8]
    return int(h, 16) / 0xFFFFFFFF


def _entropy(arr: np.ndarray) -> float:
    """Compute Shannon entropy of a normalized array."""
    p = np.abs(arr) + 1e-10
    p = p / p.sum()
    return float(-np.sum(p * np.log2(p)))
