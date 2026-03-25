from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """ML Detection service configuration."""

    # Kafka
    kafka_brokers: str = "localhost:9092"
    kafka_group_id: str = "ml-detection"
    viola_env: str = "dev"

    # Model
    contamination: float = 0.05  # Expected anomaly fraction
    baseline_window_hours: int = 24
    min_samples_for_training: int = 100
    retrain_interval_minutes: int = 60
    confidence_threshold: float = 0.75

    # Server
    metrics_port: int = 9095
    health_port: int = 8095

    # Feature extraction
    feature_dimensions: int = 12

    class Config:
        env_prefix = "ML_"
