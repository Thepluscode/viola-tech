"""ML Detection Service — Entry point.

Consumes normalized telemetry from Kafka, runs per-tenant Isolation Forest
anomaly detection, and publishes behavioral alerts.
"""

import signal
import sys
import threading

import structlog
import uvicorn
from fastapi import FastAPI
from prometheus_client import make_asgi_app, generate_latest

# Ensure package is importable
sys.path.insert(0, "/app")

from config.settings import Settings
from internal.models import AnomalyDetector
from internal.pipeline.consumer import MLPipeline

structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer(),
    ]
)
logger = structlog.get_logger()

settings = Settings()

# ── FastAPI health/metrics server ───────────────────────────────────────────

app = FastAPI(title="Viola ML Detection", version="0.1.0")
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

detector: AnomalyDetector | None = None
pipeline: MLPipeline | None = None


@app.get("/health")
def health():
    return {
        "status": "ok",
        "tenants": detector.tenant_count if detector else 0,
        "models": detector.get_model_stats() if detector else {},
    }


@app.get("/ready")
def ready():
    if detector is None:
        return {"status": "not_ready"}, 503
    return {"status": "ready"}


# ── Main ────────────────────────────────────────────────────────────────────

def main():
    global detector, pipeline

    logger.info("ml_detection_starting",
                brokers=settings.kafka_brokers,
                env=settings.viola_env,
                contamination=settings.contamination,
                threshold=settings.confidence_threshold)

    # Build topic names
    env = settings.viola_env
    input_topic = f"viola.{env}.telemetry.v1.normalized"
    alert_topic = f"viola.{env}.security.alert.v1.created"

    # Initialize detector
    detector = AnomalyDetector(
        contamination=settings.contamination,
        min_samples=settings.min_samples_for_training,
        retrain_interval=settings.retrain_interval_minutes * 60,
    )

    # Initialize pipeline
    pipeline = MLPipeline(
        brokers=settings.kafka_brokers,
        group_id=settings.kafka_group_id,
        input_topic=input_topic,
        alert_topic=alert_topic,
        detector=detector,
        confidence_threshold=settings.confidence_threshold,
    )

    # Run pipeline in background thread
    pipeline_thread = threading.Thread(target=pipeline.run, daemon=True)
    pipeline_thread.start()

    # Graceful shutdown
    def shutdown(signum, frame):
        logger.info("ml_detection_shutting_down")
        pipeline.stop()
        pipeline_thread.join(timeout=10)
        sys.exit(0)

    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)

    # Run health/metrics server
    uvicorn.run(app, host="0.0.0.0", port=settings.health_port, log_level="warning")


if __name__ == "__main__":
    main()
