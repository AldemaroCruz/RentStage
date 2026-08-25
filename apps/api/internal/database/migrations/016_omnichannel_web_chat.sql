-- RentStage v0.19.0 — omnichannel core and public web-chat foundation.
--
-- WEB_CHAT reuses the existing tenant-scoped assistant inbox. Public browser
-- sessions authenticate with an opaque bearer token; PostgreSQL stores only
-- its SHA-256 hash. Instagram and Messenger are reserved as channel values for
-- future provider adapters, but this migration does not activate them.

ALTER TABLE public_catalog_settings
  ADD COLUMN web_chat_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE assistant_conversations
  ADD COLUMN contact_email VARCHAR(320);

ALTER TABLE assistant_conversations
  DROP CONSTRAINT assistant_conversations_channel_chk,
  DROP CONSTRAINT assistant_conversations_contact_chk;

ALTER TABLE assistant_conversations
  ADD CONSTRAINT assistant_conversations_channel_chk
    CHECK (
      channel IN (
        'DEMO',
        'WHATSAPP',
        'WEB_CHAT',
        'INSTAGRAM',
        'MESSENGER'
      )
    ),
  ADD CONSTRAINT assistant_conversations_contact_chk
    CHECK (
      BTRIM(contact_name) <> ''
      AND (
        channel <> 'WHATSAPP'
        OR BTRIM(contact_phone) <> ''
      )
      AND (
        contact_email IS NULL
        OR BTRIM(contact_email) <> ''
      )
    );

ALTER TABLE assistant_messages
  DROP CONSTRAINT assistant_messages_provider_chk;

ALTER TABLE assistant_messages
  ADD CONSTRAINT assistant_messages_provider_chk
    CHECK (
      provider IN (
        'DEMO',
        'WHATSAPP',
        'WEB_CHAT',
        'INSTAGRAM',
        'MESSENGER'
      )
    );

CREATE TABLE assistant_web_chat_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL,
  token_hash CHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
  terms_version VARCHAR(40) NOT NULL,
  consent_accepted_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT assistant_web_chat_sessions_tenant_id_uniq
    UNIQUE (tenant_id, id),

  CONSTRAINT assistant_web_chat_sessions_conversation_uniq
    UNIQUE (tenant_id, conversation_id),

  CONSTRAINT assistant_web_chat_sessions_token_uniq
    UNIQUE (token_hash),

  CONSTRAINT assistant_web_chat_sessions_conversation_fk
    FOREIGN KEY (tenant_id, conversation_id)
      REFERENCES assistant_conversations(tenant_id, id)
      ON DELETE CASCADE,

  CONSTRAINT assistant_web_chat_sessions_status_chk
    CHECK (status IN ('ACTIVE', 'CLOSED', 'REVOKED')),

  CONSTRAINT assistant_web_chat_sessions_token_chk
    CHECK (token_hash ~ '^[0-9a-f]{64}$'),

  CONSTRAINT assistant_web_chat_sessions_terms_chk
    CHECK (BTRIM(terms_version) <> ''),

  CONSTRAINT assistant_web_chat_sessions_expiry_chk
    CHECK (expires_at > created_at)
);

CREATE INDEX assistant_web_chat_sessions_active_idx
  ON assistant_web_chat_sessions(tenant_id, expires_at)
  WHERE status = 'ACTIVE';

DROP TRIGGER IF EXISTS assistant_web_chat_sessions_set_updated_at
  ON assistant_web_chat_sessions;

CREATE TRIGGER assistant_web_chat_sessions_set_updated_at
BEFORE UPDATE ON assistant_web_chat_sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();