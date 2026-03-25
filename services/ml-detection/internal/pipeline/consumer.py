"""Kafka consumer that reads normalized telemetry and runs anomaly detection."""

import json
import time
from typing import Callable

import structlog
from confluent_kafka import Consumer, KafkaError, KafkaException
from prometheus_client import Counter, Histogram, Gauge

from ..models import AnomalyDetector, AnomalyResult, extract_features
from .producer import AlertProducer

logger = structlog.get_logger()

# Prometheus metrics
EVENTS_CONSUMED = Counter(
    "ml_events_consumed_total",
    "Total events consumed from Kafka",
    ["tenant_id", "event_type"],
)
ANOMALIES_DETECTED = Counter(
    "ml_anomalies_detected_total",
    "Total anomalies detected",
    ["tenant_id", "event_type", "severity"],
)
PROCESSING_TIME = Histogram(
    "ml_processing_seconds",
    "Time to process a single event",
    buckets=[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25],
)
MODEL_TENANTS = Gauge(
    "ml_model_tenants_total",
    "Number of tenant models loaded",
)


class MLPipeline:
    """Consumes normalized telemetry, runs anomaly detection, publishes alerts."""

    def __init__(
        self,
        brokers: str,
        group_id: str,
        input_topic: str,
        alert_topic: str,
        detector: AnomalyDetector,
        confidence_threshold: float = 0.75,
    ):
        self._consumer = Consumer({
            "bootstrap.servers": brokers,
            "group.id": group_id,
            "auto.offset.reset": "latest",
            "enable.auto.commit": True,
            "auto.commit.interval.ms": 5000,
        })
        self._input_topic = input_topic
        self._detector = detector
        self._confidence_threshold = confidence_threshold
        self._producer = AlertProducer(brokers, alert_topic)
        self._running = False

    def run(self) -> None:
        """Main consumer loop."""
        self._consumer.subscribe([self._input_topic])
        self._running = True

        logger.info("ml_pipeline_started",
                     topic=self._input_topic,
                     threshold=self._confidence_threshold)

        try:
            while self._running:
                msg = self._consumer.poll(timeout=1.0)
                if msg is None:
                    continue
                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        continue
                    raise KafkaException(msg.error())

                self._process_message(msg)
                MODEL_TENANTS.set(self._detector.tenant_count)
        finally:
            self._consumer.close()
            self._producer.close()

    def stop(self) -> None:
        self._running = False

    def _process_message(self, msg) -> None:
        """Process a single Kafka message."""
        start = time.monotonic()

        try:
            # Parse headers
            headers = {}
            if msg.headers():
                headers = {h[0]: h[1].decode() for h in msg.headers()}

            tenant_id = headers.get("x-tenant-id", "")
            if not tenant_id:
                return

            # Parse protobuf envelope (simplified — use JSON fallback)
            try:
                event = json.loads(msg.value())
            except (json.JSONDecodeError, TypeError):
                # Try protobuf decode in production
                return

            event_type = event.get("event_type", "unknown")
            entity_id = event.get("entity_id", "")
            payload = event.get("payload", {})
            if isinstance(payload, str):
                try:
                    payload = json.loads(payload)
                except json.JSONDecodeError:
                    payload = {}
            labels = event.get("labels", {})

            EVENTS_CONSUMED.labels(
                tenant_id=tenant_id,
                event_type=event_type,
            ).inc()

            # Extract features and score
            features = extract_features(
                tenant_id=tenant_id,
                entity_id=entity_id,
                event_type=event_type,
                payload=payload,
                labels=labels,
            )

            result = self._detector.score(features)

            # Publish alert if anomaly with sufficient confidence
            if result.is_anomaly and result.confidence >= self._confidence_threshold:
                severity = self._score_to_severity(result.anomaly_score)
                ANOMALIES_DETECTED.labels(
                    tenant_id=tenant_id,
                    event_type=event_type,
                    severity=severity,
                ).inc()

                self._producer.publish_alert(
                    tenant_id=tenant_id,
                    entity_id=entity_id,
                    event_type=event_type,
                    result=result,
                    severity=severity,
                    request_id=headers.get("x-request-id", ""),
                )

                logger.info("anomaly_detected",
                            tenant_id=tenant_id,
                            entity_id=entity_id,
                            event_type=event_type,
                            score=round(result.anomaly_score, 3),
                            confidence=round(result.confidence, 3),
                            severity=severity,
                            reason=result.reason)

        except Exception:
            logger.exception("ml_processing_error")
        finally:
            PROCESSING_TIME.observe(time.monotonic() - start)

    @staticmethod
    def _score_to_severity(score: float) -> str:
        if score >= 0.9:
            return "critical"
        elif score >= 0.7:
            return "high"
        elif score >= 0.5:
            return "med"
        return "low"
