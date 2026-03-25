create table if not exists alerts (
  tenant_id text not null,
  alert_id text not null,

  created_at timestamptz not null,
  updated_at timestamptz not null,
  status text not null,

  severity text not null,
  confidence double precision not null,
  risk_score double precision not null,

  title text not null,
  description text not null,

  mitre_tactic text null,
  mitre_technique text null,

  labels jsonb not null default '{}'::jsonb,

  assigned_to text null,
  closure_reason text null,

  request_id text null,

  primary key (tenant_id, alert_id)
);

create index if not exists idx_alerts_tenant_status_updated
  on alerts (tenant_id, status, updated_at desc);

create index if not exists idx_alerts_tenant_sev_updated
  on alerts (tenant_id, severity, updated_at desc);

create index if not exists idx_alerts_tenant_risk_updated
  on alerts (tenant_id, risk_score desc, updated_at desc);

-- Alert to entity mapping (many-to-many)
create table if not exists alert_entities (
  tenant_id text not null,
  alert_id text not null,
  entity_id text not null,
  primary key (tenant_id, alert_id, entity_id),
  foreign key (tenant_id, alert_id) references alerts(tenant_id, alert_id) on delete cascade
);

create index if not exists idx_alert_entities_entity
  on alert_entities (tenant_id, entity_id);

-- Alert to detection hit mapping (many-to-many)
create table if not exists alert_hits (
  tenant_id text not null,
  alert_id text not null,
  hit_id text not null,
  created_at timestamptz not null default now(),
  primary key (tenant_id, alert_id, hit_id),
  foreign key (tenant_id, alert_id) references alerts(tenant_id, alert_id) on delete cascade
);
