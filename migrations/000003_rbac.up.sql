-- RBAC policy table
CREATE TABLE IF NOT EXISTS rbac_policies (
  tenant_id  TEXT        NOT NULL,
  role       TEXT        NOT NULL,
  resource   TEXT        NOT NULL,
  action     TEXT        NOT NULL,
  allowed    BOOLEAN     NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, role, resource, action)
);

CREATE INDEX IF NOT EXISTS idx_rbac_policies_tenant_role
  ON rbac_policies (tenant_id, role);

-- Default RBAC policies (wildcard tenant)
INSERT INTO rbac_policies (tenant_id, role, resource, action, allowed) VALUES
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
ON CONFLICT (tenant_id, role, resource, action) DO NOTHING;
