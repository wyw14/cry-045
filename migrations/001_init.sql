CREATE TABLE IF NOT EXISTS materials (
  id text PRIMARY KEY,
  code text NOT NULL UNIQUE,
  name text NOT NULL,
  risk_class text NOT NULL,
  allowed_processes jsonb NOT NULL DEFAULT '[]',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS certificates (
  id text PRIMARY KEY,
  material_id text NOT NULL REFERENCES materials(id),
  certificate_no text NOT NULL,
  expires_at timestamptz NOT NULL,
  status text NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_certificates_expiry ON certificates(expires_at);
