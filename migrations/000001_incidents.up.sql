-- Incidents: correlated security events
CREATE TABLE IF NOT EXISTS incidents (
  tenant_id            TEXT             NOT NULL,
  incident_id          TEXT             NOT NULL,
  correlated_group_id  TEXT             NOT NULL,
  created_at           TIMESTAMPTZ      NOT NULL,
  updated_at           TIMESTAMPTZ      NOT NULL,
  status               TEXT             NOT NULL,
  severity             TEXT             NOT NULL,
  max_risk_score       DOUBLE PRECISION NOT NULL,
  max_confidence       DOUBLE PRECISION NOT NULL,
  mitre_tactic         TEXT             NULL,
  mitre_technique      TEXT             NULL,
  labels               JSONB            NOT NULL DEFAULT '{}'::jsonb,
  assigned_to          TEXT             NULL,
  closure_reason       TEXT             NULL,
  request_id           TEXT             NULL,
  alert_count          INT              NOT NULL DEFAULT 0,
  hit_count            INT              NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, incident_id)
);

CREATE INDEX IF NOT EXISTS idx_incidents_tenant_status_updated
  ON incidents (tenant_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_incidents_tenant_sev_updated
  ON incidents (tenant_id, severity, updated_at DESC);

CREATE TABLE IF NOT EXISTS incident_entities (
  tenant_id   TEXT NOT NULL,
  incident_id TEXT NOT NULL,
  entity_id   TEXT NOT NULL,
  PRIMARY KEY (tenant_id, incident_id, entity_id),
  FOREIGN KEY (tenant_id, incident_id)
    REFERENCES incidents(tenant_id, incident_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS incident_alerts (
  tenant_id   TEXT        NOT NULL,
  incident_id TEXT        NOT NULL,
  alert_id    TEXT        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, incident_id, alert_id),
  FOREIGN KEY (tenant_id, incident_id)
    REFERENCES incidents(tenant_id, incident_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS incident_hits (
  tenant_id   TEXT        NOT NULL,
  incident_id TEXT        NOT NULL,
  hit_id      TEXT        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, incident_id, hit_id),
  FOREIGN KEY (tenant_id, incident_id)
    REFERENCES incidents(tenant_id, incident_id) ON DELETE CASCADE
);
