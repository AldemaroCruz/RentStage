-- RentStage v0.15.0 — human-approved WhatsApp sales-assistant foundation.
-- DEMO is a first-class channel so the commercial flow works before a Meta
-- Business account is connected. A future provider can write to the same
-- tenant-scoped conversations without changing quote or inventory contracts.

CREATE TABLE IF NOT EXISTS assistant_conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  channel VARCHAR(16) NOT NULL DEFAULT 'DEMO',
  external_conversation_id VARCHAR(180),
  customer_id UUID,
  contact_name VARCHAR(240) NOT NULL,
  contact_phone VARCHAR(32) NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'OPEN',
  consent_status VARCHAR(16) NOT NULL DEFAULT 'DEMO',
  service_window_expires_at TIMESTAMPTZ,
  assigned_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  summary TEXT NOT NULL DEFAULT '',
  last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT assistant_conversations_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT assistant_conversations_customer_fk FOREIGN KEY (tenant_id, customer_id)
    REFERENCES customers(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT assistant_conversations_channel_chk CHECK (channel IN ('DEMO', 'WHATSAPP')),
  CONSTRAINT assistant_conversations_status_chk CHECK (status IN ('OPEN', 'HUMAN_REVIEW', 'QUOTE_DRAFTED', 'CLOSED')),
  CONSTRAINT assistant_conversations_consent_chk CHECK (consent_status IN ('DEMO', 'OPTED_IN', 'UNKNOWN', 'OPTED_OUT')),
  CONSTRAINT assistant_conversations_contact_chk CHECK (BTRIM(contact_name) <> '' AND BTRIM(contact_phone) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS assistant_conversations_external_uniq
  ON assistant_conversations(tenant_id, channel, external_conversation_id)
  WHERE external_conversation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS assistant_conversations_inbox_idx
  ON assistant_conversations(tenant_id, status, last_message_at DESC);

CREATE TABLE IF NOT EXISTS assistant_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL,
  direction VARCHAR(12) NOT NULL,
  sender_type VARCHAR(16) NOT NULL,
  provider VARCHAR(24) NOT NULL DEFAULT 'DEMO',
  external_message_id VARCHAR(180),
  body TEXT NOT NULL,
  status VARCHAR(16) NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT assistant_messages_conversation_fk FOREIGN KEY (tenant_id, conversation_id)
    REFERENCES assistant_conversations(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT assistant_messages_direction_chk CHECK (direction IN ('INBOUND', 'OUTBOUND')),
  CONSTRAINT assistant_messages_sender_chk CHECK (sender_type IN ('CUSTOMER', 'ASSISTANT', 'USER', 'SYSTEM')),
  CONSTRAINT assistant_messages_provider_chk CHECK (provider IN ('DEMO', 'WHATSAPP')),
  CONSTRAINT assistant_messages_status_chk CHECK (status IN ('RECEIVED', 'DRAFT', 'APPROVED', 'SENT', 'FAILED')),
  CONSTRAINT assistant_messages_body_chk CHECK (BTRIM(body) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS assistant_messages_external_uniq
  ON assistant_messages(tenant_id, provider, external_message_id)
  WHERE external_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS assistant_messages_timeline_idx
  ON assistant_messages(tenant_id, conversation_id, created_at, id);

CREATE TABLE IF NOT EXISTS assistant_proposals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'PROPOSED',
  provider VARCHAR(24) NOT NULL DEFAULT 'DEMO_RULES',
  event_type VARCHAR(120) NOT NULL,
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  event_location VARCHAR(500) NOT NULL,
  guest_count INTEGER NOT NULL,
  package_id UUID NOT NULL,
  package_quantity INTEGER NOT NULL DEFAULT 1,
  package_name VARCHAR(180) NOT NULL,
  package_price NUMERIC(12,2) NOT NULL,
  available BOOLEAN NOT NULL,
  recommendation TEXT NOT NULL,
  response_draft TEXT NOT NULL,
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  quote_id UUID,
  approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT assistant_proposals_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT assistant_proposals_conversation_uniq UNIQUE (tenant_id, conversation_id),
  CONSTRAINT assistant_proposals_conversation_fk FOREIGN KEY (tenant_id, conversation_id)
    REFERENCES assistant_conversations(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT assistant_proposals_package_fk FOREIGN KEY (tenant_id, package_id)
    REFERENCES packages(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT assistant_proposals_quote_fk FOREIGN KEY (tenant_id, quote_id)
    REFERENCES quotes(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT assistant_proposals_status_chk CHECK (status IN ('PROPOSED', 'APPROVED', 'QUOTE_CREATED', 'REJECTED')),
  CONSTRAINT assistant_proposals_provider_chk CHECK (provider IN ('DEMO_RULES', 'VERTEX_GEMINI')),
  CONSTRAINT assistant_proposals_dates_chk CHECK (end_at > start_at),
  CONSTRAINT assistant_proposals_guests_chk CHECK (guest_count > 0 AND guest_count <= 1000000),
  CONSTRAINT assistant_proposals_quantity_chk CHECK (package_quantity > 0 AND package_quantity <= 100),
  CONSTRAINT assistant_proposals_price_chk CHECK (package_price >= 0)
);

CREATE INDEX IF NOT EXISTS assistant_proposals_quote_idx
  ON assistant_proposals(tenant_id, quote_id)
  WHERE quote_id IS NOT NULL;

DROP TRIGGER IF EXISTS assistant_conversations_set_updated_at ON assistant_conversations;
CREATE TRIGGER assistant_conversations_set_updated_at
BEFORE UPDATE ON assistant_conversations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS assistant_proposals_set_updated_at ON assistant_proposals;
CREATE TRIGGER assistant_proposals_set_updated_at
BEFORE UPDATE ON assistant_proposals
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
