-- Response actions audit trail
CREATE TABLE IF NOT EXISTS response_actions (
  action_id    TEXT        NOT NULL,
  tenant_id    TEXT        NOT NULL,
  incident_id  TEXT        NULL,
  alert_id     TEXT        NULL,
  action_type  TEXT        NOT NULL,
  target       TEXT        NOT NULL,
  status       TEXT        NOT NULL,
  reason       TEXT        NOT NULL,
  triggered_by TEXT        NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  detail       JSONB       NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY (tenant_id, action_id)
);

CREATE INDEX IF NOT EXISTS idx_response_actions_tenant_created
  ON response_actions (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_response_actions_tenant_incident
  ON response_actions (tenant_id, incident_id)
  WHERE incident_id IS NOT NULL;
