package kafka

import "fmt"

type Topics struct {
  Env string

  TelemetryEndpointRaw string
  TelemetryIdentityRaw string
  TelemetryCloudRaw string
  TelemetryNormalized string

  DetectionHit string
  AlertCreated string
  AlertUpdated string
  IncidentUpserted string

  GraphEdgeObserved string
  GraphRiskUpdated string

  ResponseRequested string
  ResponseExecuted string
  ResponseFailed string

  AuditEvent string

  DLQIngestion string
  DLQDetection string
  DLQGraph string
  DLQResponse string
  DLQWorkers string
}

func NewTopics(env string) Topics {
  p := func(s string) string { return fmt.Sprintf("viola.%s.%s", env, s) }
  return Topics{
    Env: env,
    TelemetryEndpointRaw: p("telemetry.endpoint.v1.raw"),
    TelemetryIdentityRaw: p("telemetry.identity.v1.raw"),
    TelemetryCloudRaw: p("telemetry.cloud.v1.raw"),
    TelemetryNormalized: p("telemetry.v1.normalized"),

    DetectionHit: p("security.detection.v1.hit"),
    AlertCreated: p("security.alert.v1.created"),
    AlertUpdated: p("security.alert.v1.updated"),
    IncidentUpserted: p("security.incident.v1.upserted"),

    GraphEdgeObserved: p("graph.v1.edge.observed"),
    GraphRiskUpdated: p("graph.v1.risk.updated"),

    ResponseRequested: p("response.v1.requested"),
    ResponseExecuted: p("response.v1.executed"),
    ResponseFailed: p("response.v1.failed"),

    AuditEvent: p("audit.v1.event"),

    DLQIngestion: p("dlq.v1.ingestion"),
    DLQDetection: p("dlq.v1.detection"),
    DLQGraph: p("dlq.v1.graph"),
    DLQResponse: p("dlq.v1.response"),
    DLQWorkers: p("dlq.v1.workers"),
  }
}
