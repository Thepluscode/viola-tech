# Kafka Topics & Contracts (MVP) — Viola Technologies

See conversation-derived baseline. This repo implements:
- raw telemetry topics
- normalized telemetry topic
- detection hits
- alerts
- incidents
- audit + DLQ schemas (protobuf)

All topics are generated via `shared/go/kafka.NewTopics(env)`.
