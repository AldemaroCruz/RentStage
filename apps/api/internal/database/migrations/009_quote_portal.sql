-- RentStage v0.10.0 — customer-facing quote portal.
-- Public links use bearer tokens whose raw value is never persisted.

CREATE TABLE IF NOT EXISTS quote_portal_settings (
  tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  headline VARCHAR(180) NOT NULL DEFAULT 'Tu cotización está lista',
  introduction TEXT NOT NULL DEFAULT 'Revisa los detalles, las fechas y los términos antes de responder.',
  accent_color VARCHAR(7) NOT NULL DEFAULT '#6558e8',
  default_validity_days INTEGER NOT NULL DEFAULT 7,
  allow_rejection BOOLEAN NOT NULL DEFAULT TRUE,
  require_response_name BOOLEAN NOT NULL DEFAULT TRUE,
  acceptance_terms_text TEXT NOT NULL DEFAULT 'Al aceptar confirmo que revisé la cotización, sus fechas, precios y condiciones comerciales.',
  acceptance_terms_version VARCHAR(40) NOT NULL DEFAULT '1.0',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT quote_portal_settings_color_chk CHECK (accent_color ~ '^#[0-9A-Fa-f]{6}$'),
  CONSTRAINT quote_portal_settings_validity_chk CHECK (default_validity_days BETWEEN 1 AND 60),
  CONSTRAINT quote_portal_settings_headline_chk CHECK (char_length(headline) BETWEEN 1 AND 180),
  CONSTRAINT quote_portal_settings_intro_chk CHECK (char_length(introduction) <= 2000),
  CONSTRAINT quote_portal_settings_terms_chk CHECK (char_length(acceptance_terms_text) BETWEEN 1 AND 12000),
  CONSTRAINT quote_portal_settings_version_chk CHECK (char_length(acceptance_terms_version) BETWEEN 1 AND 40)
);

CREATE TABLE IF NOT EXISTS quote_portals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  quote_id UUID NOT NULL,
  token_hash CHAR(64) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  revision INTEGER NOT NULL DEFAULT 1,
  expires_at TIMESTAMPTZ NOT NULL,
  headline VARCHAR(180) NOT NULL,
  introduction TEXT NOT NULL,
  accent_color VARCHAR(7) NOT NULL,
  allow_rejection BOOLEAN NOT NULL,
  require_response_name BOOLEAN NOT NULL,
  terms_text TEXT NOT NULL,
  terms_version VARCHAR(40) NOT NULL,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  last_viewed_at TIMESTAMPTZ,
  view_count INTEGER NOT NULL DEFAULT 0,
  decision_at TIMESTAMPTZ,
  decision_source VARCHAR(20),
  response_name VARCHAR(180),
  response_email VARCHAR(320),
  rejection_reason TEXT,
  origin_hash CHAR(64),
  user_agent VARCHAR(500),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT quote_portals_quote_fk
    FOREIGN KEY (tenant_id, quote_id) REFERENCES quotes(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT quote_portals_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT quote_portals_tenant_quote_uniq UNIQUE (tenant_id, quote_id),
  CONSTRAINT quote_portals_token_hash_uniq UNIQUE (token_hash),
  CONSTRAINT quote_portals_status_chk CHECK (status IN ('ACTIVE', 'ACCEPTED', 'REJECTED', 'REVOKED', 'EXPIRED')),
  CONSTRAINT quote_portals_revision_chk CHECK (revision >= 1),
  CONSTRAINT quote_portals_color_chk CHECK (accent_color ~ '^#[0-9A-Fa-f]{6}$'),
  CONSTRAINT quote_portals_view_count_chk CHECK (view_count >= 0),
  CONSTRAINT quote_portals_decision_source_chk CHECK (decision_source IS NULL OR decision_source IN ('CUSTOMER', 'ADMIN', 'SYSTEM')),
  CONSTRAINT quote_portals_response_name_chk CHECK (response_name IS NULL OR char_length(response_name) BETWEEN 1 AND 180),
  CONSTRAINT quote_portals_response_email_chk CHECK (response_email IS NULL OR char_length(response_email) <= 320),
  CONSTRAINT quote_portals_rejection_reason_chk CHECK (rejection_reason IS NULL OR char_length(rejection_reason) <= 2000)
);

CREATE TABLE IF NOT EXISTS quote_portal_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  portal_id UUID NOT NULL,
  quote_id UUID NOT NULL,
  event_type VARCHAR(40) NOT NULL,
  actor_type VARCHAR(20) NOT NULL,
  actor_id VARCHAR(180) NOT NULL DEFAULT '',
  origin_hash CHAR(64),
  user_agent VARCHAR(500),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT quote_portal_events_portal_fk
    FOREIGN KEY (tenant_id, portal_id) REFERENCES quote_portals(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT quote_portal_events_quote_fk
    FOREIGN KEY (tenant_id, quote_id) REFERENCES quotes(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT quote_portal_events_type_chk CHECK (event_type IN (
    'CREATED', 'REISSUED', 'VIEWED', 'ACCEPTED', 'REJECTED',
    'REVOKED', 'EXPIRED', 'ACCEPTANCE_BLOCKED'
  )),
  CONSTRAINT quote_portal_events_actor_chk CHECK (actor_type IN ('CUSTOMER', 'USER', 'SYSTEM', 'API'))
);

CREATE INDEX IF NOT EXISTS quote_portals_tenant_status_idx
  ON quote_portals (tenant_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS quote_portals_expiration_idx
  ON quote_portals (status, expires_at) WHERE status = 'ACTIVE';
CREATE INDEX IF NOT EXISTS quote_portal_events_portal_idx
  ON quote_portal_events (tenant_id, portal_id, created_at DESC);
CREATE INDEX IF NOT EXISTS quote_portal_events_quote_idx
  ON quote_portal_events (tenant_id, quote_id, created_at DESC);

DROP TRIGGER IF EXISTS set_quote_portal_settings_updated_at ON quote_portal_settings;
CREATE TRIGGER set_quote_portal_settings_updated_at
BEFORE UPDATE ON quote_portal_settings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_quote_portals_updated_at ON quote_portals;
CREATE TRIGGER set_quote_portals_updated_at
BEFORE UPDATE ON quote_portals
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO quote_portal_settings (tenant_id)
SELECT id FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;

-- Keep customer portals aligned when an administrator changes a quote status
-- through the protected back office. Customer responses overwrite the generic
-- ADMIN evidence in the same transaction with their explicit details.
CREATE OR REPLACE FUNCTION sync_quote_portal_from_quote_status()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.status IS DISTINCT FROM OLD.status THEN
    IF NEW.status = 'ACCEPTED' THEN
      UPDATE quote_portals
      SET status = 'ACCEPTED',
          decision_at = COALESCE(decision_at, NOW()),
          decision_source = COALESCE(decision_source, 'ADMIN')
      WHERE tenant_id = NEW.tenant_id
        AND quote_id = NEW.id
        AND status = 'ACTIVE';
    ELSIF NEW.status = 'REJECTED' THEN
      UPDATE quote_portals
      SET status = 'REJECTED',
          decision_at = COALESCE(decision_at, NOW()),
          decision_source = COALESCE(decision_source, 'ADMIN')
      WHERE tenant_id = NEW.tenant_id
        AND quote_id = NEW.id
        AND status = 'ACTIVE';
    ELSIF NEW.status = 'CANCELLED' THEN
      UPDATE quote_portals
      SET status = 'REVOKED',
          decision_at = COALESCE(decision_at, NOW()),
          decision_source = COALESCE(decision_source, 'ADMIN')
      WHERE tenant_id = NEW.tenant_id
        AND quote_id = NEW.id
        AND status = 'ACTIVE';
    ELSIF NEW.status = 'EXPIRED' THEN
      UPDATE quote_portals
      SET status = 'EXPIRED',
          decision_at = COALESCE(decision_at, NOW()),
          decision_source = COALESCE(decision_source, 'SYSTEM')
      WHERE tenant_id = NEW.tenant_id
        AND quote_id = NEW.id
        AND status = 'ACTIVE';
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS sync_quote_portal_status ON quotes;
CREATE TRIGGER sync_quote_portal_status
AFTER UPDATE OF status ON quotes
FOR EACH ROW EXECUTE FUNCTION sync_quote_portal_from_quote_status();
