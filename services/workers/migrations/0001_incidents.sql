-- 0001_incidents.sql (workers service)
--
-- Incident materialisation tables.  These are shared with the gateway-api
-- service (same Postgres database).  This file is provided for local
-- development bootstrapping; in production apply migrations via the
-- gateway-api/migrations directory only to avoid running DDL twice.
--
-- Design notes:
--   - alert_count is NOT maintained by triggers; it is recomputed from the
--     incident_alerts link table after each idempotent INSERT, making the
--     worker safe to replay under Kafka at-least-once delivery.
--   - severity is resolved in Go application code (workers/internal/incident/
--     store.go: maxSeverity()) — no greatest_severity() Postgres UDF.

create table if not exists incidents (
  tenant_id            text        not null,
  incident_id          text        not null,
  correlated_group_id  text        not null,

  created_at           timestamptz not null,
  updated_at           timestamptz not null,
  status               text        not null,

  severity             text        not null,
  max_risk_score       double precision not null,
  max_confidence       double precision not null,

  mitre_tactic         text        null,
  mitre_technique      text        null,

  labels               jsonb       not null default '{}'::jsonb,

  assigned_to          text        null,
  closure_reason       text        null,
  request_id           text        null,

  alert_count          int         not null default 0,
  hit_count            int         not null default 0,

  primary key (tenant_id, incident_id)
);

create index if not exists idx_incidents_tenant_status_updated
  on incidents (tenant_id, status, updated_at desc);

create index if not exists idx_incidents_tenant_sev_updated
  on incidents (tenant_id, severity, updated_at desc);

-- Incident → entity associations (many-to-many, idempotent via pk).
create table if not exists incident_entities (
  tenant_id   text not null,
  incident_id text not null,
  entity_id   text not null,
  primary key (tenant_id, incident_id, entity_id),
  foreign key (tenant_id, incident_id)
    references incidents(tenant_id, incident_id) on delete cascade
);

-- Incident → alert associations.
-- INSERT here is idempotent (pk); alert_count is recomputed as COUNT(*).
create table if not exists incident_alerts (
  tenant_id   text        not null,
  incident_id text        not null,
  alert_id    text        not null,
  created_at  timestamptz not null default now(),
  primary key (tenant_id, incident_id, alert_id),
  foreign key (tenant_id, incident_id)
    references incidents(tenant_id, incident_id) on delete cascade
);

-- Incident → detection-hit associations.
create table if not exists incident_hits (
  tenant_id   text        not null,
  incident_id text        not null,
  hit_id      text        not null,
  created_at  timestamptz not null default now(),
  primary key (tenant_id, incident_id, hit_id),
  foreign key (tenant_id, incident_id)
    references incidents(tenant_id, incident_id) on delete cascade
);
