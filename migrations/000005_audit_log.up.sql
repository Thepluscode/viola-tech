-- Audit log for compliance reporting (optional persistent store alongside Kafka)
CREATE TABLE IF NOT EXISTS audit_log (
  id           BIGSERIAL   NOT NULL,
  tenant_id    TEXT        NOT NULL,
  timestamp    TIMESTAMPTZ NOT NULL DEFAULT now(),
  actor_type   TEXT        NOT NULL,
  actor_id     TEXT        NOT NULL,
  actor_ip     TEXT        NULL,
  resource_type TEXT       NOT NULL,
  resource_id  TEXT        NOT NULL,
  action       TEXT        NOT NULL,
  outcome      TEXT        NOT NULL,
  reason       TEXT        NULL,
  metadata     JSONB       NOT NULL DEFAULT '{}'::jsonb,
  request_id   TEXT        NULL,
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_audit_log_tenant_time
  ON audit_log (tenant_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_audit_log_actor
  ON audit_log (tenant_id, actor_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_audit_log_resource
  ON audit_log (tenant_id, resource_type, resource_id, timestamp DESC);

-- Partition hint: in production, partition by tenant_id or timestamp range
-- for efficient retention management and query performance.
