-- ── Viola XDR — Database Schema ─────────────────────────────────────────────
-- Applied automatically by Postgres on first container start.
-- Safe to re-run: all statements use IF NOT EXISTS / ON CONFLICT DO NOTHING.
-- ─────────────────────────────────────────────────────────────────────────────

-- ── Incidents ────────────────────────────────────────────────────────────────

create table if not exists incidents (
  tenant_id            text             not null,
  incident_id          text             not null,
  correlated_group_id  text             not null,
  created_at           timestamptz      not null,
  updated_at           timestamptz      not null,
  status               text             not null,
  severity             text             not null,
  max_risk_score       double precision not null,
  max_confidence       double precision not null,
  mitre_tactic         text             null,
  mitre_technique      text             null,
  labels               jsonb            not null default '{}'::jsonb,
  assigned_to          text             null,
  closure_reason       text             null,
  request_id           text             null,
  alert_count          int              not null default 0,
  hit_count            int              not null default 0,
  primary key (tenant_id, incident_id)
);

create index if not exists idx_incidents_tenant_status_updated
  on incidents (tenant_id, status, updated_at desc);

create index if not exists idx_incidents_tenant_sev_updated
  on incidents (tenant_id, severity, updated_at desc);

create table if not exists incident_entities (
  tenant_id   text not null,
  incident_id text not null,
  entity_id   text not null,
  primary key (tenant_id, incident_id, entity_id),
  foreign key (tenant_id, incident_id)
    references incidents(tenant_id, incident_id) on delete cascade
);

create table if not exists incident_alerts (
  tenant_id   text        not null,
  incident_id text        not null,
  alert_id    text        not null,
  created_at  timestamptz not null default now(),
  primary key (tenant_id, incident_id, alert_id),
  foreign key (tenant_id, incident_id)
    references incidents(tenant_id, incident_id) on delete cascade
);

create table if not exists incident_hits (
  tenant_id   text        not null,
  incident_id text        not null,
  hit_id      text        not null,
  created_at  timestamptz not null default now(),
  primary key (tenant_id, incident_id, hit_id),
  foreign key (tenant_id, incident_id)
    references incidents(tenant_id, incident_id) on delete cascade
);

-- ── Alerts ───────────────────────────────────────────────────────────────────

create table if not exists alerts (
  tenant_id       text             not null,
  alert_id        text             not null,
  created_at      timestamptz      not null,
  updated_at      timestamptz      not null,
  status          text             not null,
  severity        text             not null,
  confidence      double precision not null,
  risk_score      double precision not null,
  title           text             not null,
  description     text             not null,
  mitre_tactic    text             null,
  mitre_technique text             null,
  labels          jsonb            not null default '{}'::jsonb,
  assigned_to     text             null,
  closure_reason  text             null,
  request_id      text             null,
  primary key (tenant_id, alert_id)
);

create index if not exists idx_alerts_tenant_status_updated
  on alerts (tenant_id, status, updated_at desc);

create index if not exists idx_alerts_tenant_sev_updated
  on alerts (tenant_id, severity, updated_at desc);

create index if not exists idx_alerts_tenant_risk_updated
  on alerts (tenant_id, risk_score desc, updated_at desc);

create table if not exists alert_entities (
  tenant_id text not null,
  alert_id  text not null,
  entity_id text not null,
  primary key (tenant_id, alert_id, entity_id),
  foreign key (tenant_id, alert_id)
    references alerts(tenant_id, alert_id) on delete cascade
);

create index if not exists idx_alert_entities_entity
  on alert_entities (tenant_id, entity_id);

create table if not exists alert_hits (
  tenant_id  text        not null,
  alert_id   text        not null,
  hit_id     text        not null,
  created_at timestamptz not null default now(),
  primary key (tenant_id, alert_id, hit_id),
  foreign key (tenant_id, alert_id)
    references alerts(tenant_id, alert_id) on delete cascade
);

-- ── RBAC ─────────────────────────────────────────────────────────────────────

create table if not exists rbac_policies (
  tenant_id  text        not null,
  role       text        not null,
  resource   text        not null,
  action     text        not null,
  allowed    boolean     not null default true,
  created_at timestamptz not null default now(),
  primary key (tenant_id, role, resource, action)
);

insert into rbac_policies (tenant_id, role, resource, action, allowed) values
  ('*', 'admin',   'incidents', 'read',   true),
  ('*', 'admin',   'incidents', 'update', true),
  ('*', 'admin',   'alerts',    'read',   true),
  ('*', 'admin',   'alerts',    'update', true),
  ('*', 'analyst', 'incidents', 'read',   true),
  ('*', 'analyst', 'incidents', 'update', true),
  ('*', 'analyst', 'alerts',    'read',   true),
  ('*', 'analyst', 'alerts',    'update', true),
  ('*', 'viewer',  'incidents', 'read',   true),
  ('*', 'viewer',  'incidents', 'update', false),
  ('*', 'viewer',  'alerts',    'read',   true),
  ('*', 'viewer',  'alerts',    'update', false)
on conflict (tenant_id, role, resource, action) do nothing;

create index if not exists idx_rbac_policies_tenant_role
  on rbac_policies (tenant_id, role);

-- ── Response Actions ──────────────────────────────────────────────────────────
-- Audit trail for every automated or analyst-triggered response action.

create table if not exists response_actions (
  action_id    text        not null,
  tenant_id    text        not null,
  incident_id  text        null,  -- optional link to incident
  alert_id     text        null,  -- optional link to alert
  action_type  text        not null, -- "isolate_host" | "block_ip" | "kill_process" | ...
  target       text        not null, -- host, IP, process name, etc.
  status       text        not null, -- "pending" | "success" | "failed"
  reason       text        not null, -- human readable explanation
  triggered_by text        not null, -- "auto" | sub (user ID)
  created_at   timestamptz not null default now(),
  updated_at   timestamptz not null default now(),
  detail       jsonb       not null default '{}'::jsonb,
  primary key (tenant_id, action_id)
);

create index if not exists idx_response_actions_tenant_created
  on response_actions (tenant_id, created_at desc);

create index if not exists idx_response_actions_tenant_incident
  on response_actions (tenant_id, incident_id)
  where incident_id is not null;

-- ── Audit Log ───────────────────────────────────────────────────────────────
-- Persistent audit trail for compliance reporting (complements Kafka audit topic).

create table if not exists audit_log (
  id             bigserial   not null,
  tenant_id      text        not null,
  timestamp      timestamptz not null default now(),
  actor_type     text        not null,
  actor_id       text        not null,
  actor_ip       text        null,
  resource_type  text        not null,
  resource_id    text        not null,
  action         text        not null,
  outcome        text        not null,
  reason         text        null,
  metadata       jsonb       not null default '{}'::jsonb,
  request_id     text        null,
  primary key (id)
);

create index if not exists idx_audit_log_tenant_time
  on audit_log (tenant_id, timestamp desc);

create index if not exists idx_audit_log_actor
  on audit_log (tenant_id, actor_id, timestamp desc);

create index if not exists idx_audit_log_resource
  on audit_log (tenant_id, resource_type, resource_id, timestamp desc);

-- ── Schema Migrations Tracking ──────────────────────────────────────────────

create table if not exists schema_migrations (
  version    int         not null,
  dirty      bool        not null default false,
  applied_at timestamptz not null default now(),
  primary key (version)
);
