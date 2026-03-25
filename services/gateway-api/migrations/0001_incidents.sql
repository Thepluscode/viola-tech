create table if not exists incidents (
  tenant_id text not null,
  incident_id text not null,
  correlated_group_id text not null,

  created_at timestamptz not null,
  updated_at timestamptz not null,
  status text not null,

  severity text not null,
  max_risk_score double precision not null,
  max_confidence double precision not null,

  mitre_tactic text null,
  mitre_technique text null,

  labels jsonb not null default '{}'::jsonb,

  assigned_to text null,
  closure_reason text null,

  request_id text null,

  alert_count int not null default 0,
  hit_count int not null default 0,

  primary key (tenant_id, incident_id)
);

create index if not exists idx_incidents_tenant_status_updated
  on incidents (tenant_id, status, updated_at desc);

create index if not exists idx_incidents_tenant_sev_updated
  on incidents (tenant_id, severity, updated_at desc);

create table if not exists incident_entities (
  tenant_id text not null,
  incident_id text not null,
  entity_id text not null,
  primary key (tenant_id, incident_id, entity_id),
  foreign key (tenant_id, incident_id) references incidents(tenant_id, incident_id) on delete cascade
);

create table if not exists incident_alerts (
  tenant_id text not null,
  incident_id text not null,
  alert_id text not null,
  created_at timestamptz not null default now(),
  primary key (tenant_id, incident_id, alert_id),
  foreign key (tenant_id, incident_id) references incidents(tenant_id, incident_id) on delete cascade
);

create table if not exists incident_hits (
  tenant_id text not null,
  incident_id text not null,
  hit_id text not null,
  created_at timestamptz not null default now(),
  primary key (tenant_id, incident_id, hit_id),
  foreign key (tenant_id, incident_id) references incidents(tenant_id, incident_id) on delete cascade
);
