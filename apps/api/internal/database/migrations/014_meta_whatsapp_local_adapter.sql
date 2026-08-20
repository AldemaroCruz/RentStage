-- RentStage v0.18.0 — tenant-scoped WhatsApp channel connections.
-- Public Meta identifiers are stored here; access tokens and app secrets stay
-- in environment-specific secret stores and are never persisted in PostgreSQL.

CREATE TABLE IF NOT EXISTS assistant_channel_connections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  provider VARCHAR(24) NOT NULL,
  mode VARCHAR(20) NOT NULL,
  phone_number_id VARCHAR(180) NOT NULL,
  waba_id VARCHAR(180) NOT NULL,
  display_phone_number VARCHAR(32),
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT assistant_channel_connections_tenant_provider_uniq UNIQUE (tenant_id, provider),
  CONSTRAINT assistant_channel_connections_phone_uniq UNIQUE (provider, phone_number_id),
  CONSTRAINT assistant_channel_connections_provider_chk CHECK (provider = 'WHATSAPP'),
  CONSTRAINT assistant_channel_connections_mode_chk CHECK (mode IN ('LOCAL_MOCK', 'META_CLOUD')),
  CONSTRAINT assistant_channel_connections_ids_chk CHECK (
    BTRIM(phone_number_id) <> '' AND BTRIM(waba_id) <> ''
  )
);

CREATE INDEX IF NOT EXISTS assistant_channel_connections_enabled_idx
  ON assistant_channel_connections(provider, phone_number_id)
  WHERE enabled;

DROP TRIGGER IF EXISTS assistant_channel_connections_set_updated_at
  ON assistant_channel_connections;
CREATE TRIGGER assistant_channel_connections_set_updated_at
BEFORE UPDATE ON assistant_channel_connections
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
