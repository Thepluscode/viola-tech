-- ── Viola XDR — Development Seed Data ───────────────────────────────────────
-- Creates a test tenant so the smoke test has something to query.
-- Tenant ID: tenant-dev-001
-- ─────────────────────────────────────────────────────────────────────────────

-- Tenant-specific RBAC policies
insert into rbac_policies (tenant_id, role, resource, action, allowed) values
  ('tenant-dev-001', 'admin',   'incidents', 'read',   true),
  ('tenant-dev-001', 'admin',   'incidents', 'update', true),
  ('tenant-dev-001', 'admin',   'alerts',    'read',   true),
  ('tenant-dev-001', 'admin',   'alerts',    'update', true)
on conflict (tenant_id, role, resource, action) do nothing;
