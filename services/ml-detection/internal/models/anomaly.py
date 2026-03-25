"""Per-tenant anomaly detection using Isolation Forest with online retraining."""

import threading
import time
from collections import defaultdict, deque
from dataclasses import dataclass
from typing import Optional

import numpy as np
import structlog
from sklearn.ensemble import IsolationForest

from .features import TelemetryFeatures

logger = structlog.get_logger()


@dataclass
class AnomalyResult:
    """Result of anomaly scoring."""

    tenant_id: str
    entity_id: str
    event_type: str
    is_anomaly: bool
    anomaly_score: float  # 0.0 (normal) to 1.0 (highly anomalous)
    confidence: float  # Model confidence based on training data size
    reason: str
    features: np.ndarray


class TenantModel:
    """Per-tenant Isolation Forest model with rolling training window."""

    def __init__(self, tenant_id: str, contamination: float = 0.05,
                 window_size: int = 10_000, min_samples: int = 100):
        self.tenant_id = tenant_id
        self.contamination = contamination
        self.min_samples = min_samples
        self._buffer: deque[np.ndarray] = deque(maxlen=window_size)
        self._model: Optional[IsolationForest] = None
        self._trained_at: float = 0
        self._sample_count: int = 0
        self._lock = threading.Lock()

    @property
    def is_trained(self) -> bool:
        return self._model is not None

    @property
    def sample_count(self) -> int:
        return self._sample_count

    def add_sample(self, features: np.ndarray) -> None:
        """Add a feature vector to the training buffer."""
        self._buffer.append(features)
        self._sample_count += 1

    def train(self) -> bool:
        """Train or retrain the model from the current buffer."""
        if len(self._buffer) < self.min_samples:
            return False

        X = np.array(list(self._buffer))

        with self._lock:
            model = IsolationForest(
                contamination=self.contamination,
                n_estimators=200,
                max_samples="auto",
                random_state=42,
                n_jobs=1,
            )
            model.fit(X)
            self._model = model
            self._trained_at = time.time()

        logger.info("model_trained",
                     tenant_id=self.tenant_id,
                     samples=len(X),
                     contamination=self.contamination)
        return True

    def score(self, event: TelemetryFeatures) -> AnomalyResult:
        """Score a single event for anomaly detection."""
        self.add_sample(event.features)

        if not self.is_trained:
            return AnomalyResult(
                tenant_id=event.tenant_id,
                entity_id=event.entity_id,
                event_type=event.event_type,
                is_anomaly=False,
                anomaly_score=0.0,
                confidence=0.0,
                reason="model_not_trained",
                features=event.features,
            )

        with self._lock:
            X = event.features.reshape(1, -1)
            raw_score = self._model.decision_function(X)[0]
            prediction = self._model.predict(X)[0]

        # Convert raw score to 0-1 range (lower raw = more anomalous)
        # decision_function returns negative for anomalies
        anomaly_score = max(0.0, min(1.0, -raw_score))

        # Confidence scales with training data size
        confidence = min(1.0, self._sample_count / (self.min_samples * 10))

        is_anomaly = prediction == -1

        reason = "normal"
        if is_anomaly:
            reason = self._explain_anomaly(event)

        return AnomalyResult(
            tenant_id=event.tenant_id,
            entity_id=event.entity_id,
            event_type=event.event_type,
            is_anomaly=is_anomaly,
            anomaly_score=anomaly_score,
            confidence=confidence,
            reason=reason,
            features=event.features,
        )

    def _explain_anomaly(self, event: TelemetryFeatures) -> str:
        """Generate a human-readable explanation for the anomaly."""
        if len(self._buffer) == 0:
            return "anomaly_detected"

        X = np.array(list(self._buffer))
        means = X.mean(axis=0)
        stds = X.std(axis=0) + 1e-10

        # Find features that deviate most from baseline
        z_scores = np.abs((event.features - means) / stds)
        top_features = np.argsort(z_scores)[::-1][:3]

        feature_names = [
            "event_type", "payload_size", "field_count",
            "network_fields", "process_fields", "auth_fields",
            "hour_of_day", "numeric_entropy", "string_length",
            "label_count", "source", "cmdline_length",
        ]

        deviations = []
        for idx in top_features:
            if z_scores[idx] > 2.0:
                deviations.append(f"{feature_names[idx]}(z={z_scores[idx]:.1f})")

        if deviations:
            return "anomaly:" + ",".join(deviations)
        return "anomaly_detected"


class AnomalyDetector:
    """Multi-tenant anomaly detector managing per-tenant models."""

    def __init__(self, contamination: float = 0.05,
                 min_samples: int = 100,
                 retrain_interval: float = 3600):
        self._contamination = contamination
        self._min_samples = min_samples
        self._retrain_interval = retrain_interval
        self._models: dict[str, TenantModel] = {}
        self._lock = threading.Lock()

    def score(self, event: TelemetryFeatures) -> AnomalyResult:
        """Score a telemetry event for anomaly detection."""
        model = self._get_or_create_model(event.tenant_id)
        result = model.score(event)

        # Check if retraining is needed
        if (not model.is_trained and model.sample_count >= self._min_samples) or \
           (model.is_trained and time.time() - model._trained_at > self._retrain_interval):
            # Train in background to avoid blocking
            threading.Thread(target=model.train, daemon=True).start()

        return result

    def _get_or_create_model(self, tenant_id: str) -> TenantModel:
        with self._lock:
            if tenant_id not in self._models:
                self._models[tenant_id] = TenantModel(
                    tenant_id=tenant_id,
                    contamination=self._contamination,
                    min_samples=self._min_samples,
                )
            return self._models[tenant_id]

    @property
    def tenant_count(self) -> int:
        return len(self._models)

    def get_model_stats(self) -> dict:
        """Return training statistics for all tenant models."""
        stats = {}
        for tid, model in self._models.items():
            stats[tid] = {
                "is_trained": model.is_trained,
                "sample_count": model.sample_count,
                "buffer_size": len(model._buffer),
            }
        return stats
