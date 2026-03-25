-- RBAC policy table
create table if not exists rbac_policies (
  tenant_id text not null,
  role text not null,
  resource text not null,  -- incidents|alerts
  action text not null,    -- read|update
  allowed boolean not null default true,
  created_at timestamptz not null default now(),
  primary key (tenant_id, role, resource, action)
);

-- Default policies for common roles
insert into rbac_policies (tenant_id, role, resource, action, allowed) values
  ('*', 'admin', 'incidents', 'read', true),
  ('*', 'admin', 'incidents', 'update', true),
  ('*', 'admin', 'alerts', 'read', true),
  ('*', 'admin', 'alerts', 'update', true),
  ('*', 'analyst', 'incidents', 'read', true),
  ('*', 'analyst', 'incidents', 'update', true),
  ('*', 'analyst', 'alerts', 'read', true),
  ('*', 'analyst', 'alerts', 'update', true),
  ('*', 'viewer', 'incidents', 'read', true),
  ('*', 'viewer', 'incidents', 'update', false),
  ('*', 'viewer', 'alerts', 'read', true),
  ('*', 'viewer', 'alerts', 'update', false)
on conflict (tenant_id, role, resource, action) do nothing;

create index if not exists idx_rbac_policies_tenant_role
  on rbac_policies (tenant_id, role);
