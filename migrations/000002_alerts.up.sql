-- Alerts: individual detection alerts
CREATE TABLE IF NOT EXISTS alerts (
  tenant_id       TEXT             NOT NULL,
  alert_id        TEXT             NOT NULL,
  created_at      TIMESTAMPTZ      NOT NULL,
  updated_at      TIMESTAMPTZ      NOT NULL,
  status          TEXT             NOT NULL,
  severity        TEXT             NOT NULL,
  confidence      DOUBLE PRECISION NOT NULL,
  risk_score      DOUBLE PRECISION NOT NULL,
  title           TEXT             NOT NULL,
  description     TEXT             NOT NULL,
  mitre_tactic    TEXT             NULL,
  mitre_technique TEXT             NULL,
  labels          JSONB            NOT NULL DEFAULT '{}'::jsonb,
  assigned_to     TEXT             NULL,
  closure_reason  TEXT             NULL,
  request_id      TEXT             NULL,
  PRIMARY KEY (tenant_id, alert_id)
);

CREATE INDEX IF NOT EXISTS idx_alerts_tenant_status_updated
  ON alerts (tenant_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_alerts_tenant_sev_updated
  ON alerts (tenant_id, severity, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_alerts_tenant_risk_updated
  ON alerts (tenant_id, risk_score DESC, updated_at DESC);

CREATE TABLE IF NOT EXISTS alert_entities (
  tenant_id TEXT NOT NULL,
  alert_id  TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  PRIMARY KEY (tenant_id, alert_id, entity_id),
  FOREIGN KEY (tenant_id, alert_id)
    REFERENCES alerts(tenant_id, alert_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_alert_entities_entity
  ON alert_entities (tenant_id, entity_id);

CREATE TABLE IF NOT EXISTS alert_hits (
  tenant_id  TEXT        NOT NULL,
  alert_id   TEXT        NOT NULL,
  hit_id     TEXT        NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, alert_id, hit_id),
  FOREIGN KEY (tenant_id, alert_id)
    REFERENCES alerts(tenant_id, alert_id) ON DELETE CASCADE
);
