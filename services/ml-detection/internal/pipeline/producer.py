"""Kafka producer for ML-generated alerts."""

import json
import time
import uuid

import structlog
from confluent_kafka import Producer

from ..models import AnomalyResult

logger = structlog.get_logger()


class AlertProducer:
    """Publishes anomaly-based alerts to the alert topic."""

    def __init__(self, brokers: str, topic: str):
        self._producer = Producer({
            "bootstrap.servers": brokers,
            "linger.ms": 50,
            "batch.num.messages": 100,
        })
        self._topic = topic

    def publish_alert(
        self,
        tenant_id: str,
        entity_id: str,
        event_type: str,
        result: AnomalyResult,
        severity: str,
        request_id: str = "",
    ) -> None:
        """Publish an ML-generated alert to Kafka."""
        alert_id = f"ml-{uuid.uuid4().hex[:12]}"
        now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())

        alert = {
            "tenant_id": tenant_id,
            "alert_id": alert_id,
            "created_at": now,
            "updated_at": now,
            "status": "open",
            "severity": severity,
            "confidence": round(result.confidence, 4),
            "risk_score": round(result.anomaly_score * 100, 2),
            "title": f"ML Anomaly: {event_type} on {entity_id}",
            "description": f"Behavioral anomaly detected. Reason: {result.reason}. "
                          f"Score: {result.anomaly_score:.3f}, "
                          f"Confidence: {result.confidence:.3f}",
            "entity_ids": [entity_id],
            "detection_hit_ids": [],
            "mitre_tactic": self._infer_tactic(event_type),
            "mitre_technique": "",
            "labels": {
                "detection_source": "ml-anomaly",
                "event_type": event_type,
                "anomaly_reason": result.reason,
            },
            "assigned_to": "",
            "closure_reason": "",
            "request_id": request_id,
            "correlated_group_id": f"ml-{tenant_id}-{entity_id}",
        }

        headers = [
            ("x-tenant-id", tenant_id.encode()),
            ("x-request-id", (request_id or alert_id).encode()),
            ("x-source", b"ml-detection"),
            ("x-schema", b"viola.security.v1.Alert"),
            ("x-emitted-at", now.encode()),
        ]

        self._producer.produce(
            topic=self._topic,
            key=f"{tenant_id}:{alert_id}".encode(),
            value=json.dumps(alert).encode(),
            headers=headers,
            callback=self._delivery_callback,
        )
        self._producer.poll(0)

    def close(self) -> None:
        self._producer.flush(timeout=10)

    @staticmethod
    def _delivery_callback(err, msg):
        if err:
            logger.error("alert_publish_failed", error=str(err))

    @staticmethod
    def _infer_tactic(event_type: str) -> str:
        """Map event types to MITRE ATT&CK tactics."""
        mapping = {
            "process_start": "Execution",
            "process_access": "Credential Access",
            "network_connection": "Command and Control",
            "file_create": "Persistence",
            "file_modify": "Defense Evasion",
            "registry_modify": "Persistence",
            "dns_query": "Command and Control",
            "auth_failure": "Credential Access",
            "privilege_escalation": "Privilege Escalation",
        }
        return mapping.get(event_type, "")
