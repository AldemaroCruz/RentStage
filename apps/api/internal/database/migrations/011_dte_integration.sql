-- RentStage v0.12.0 — El Salvador DTE integration foundation.
-- The local MOCK provider exercises the complete lifecycle without calling a
-- tax authority. MH_HTTP remains disabled until the tenant supplies official
-- DGII onboarding endpoints and credentials through secret references.

ALTER TABLE customers
  ADD COLUMN IF NOT EXISTS document_type_code VARCHAR(4) NOT NULL DEFAULT '36',
  ADD COLUMN IF NOT EXISTS trade_name VARCHAR(240) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS economic_activity VARCHAR(240) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS economic_activity_code VARCHAR(12) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS department_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS municipality_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS district_code VARCHAR(8) NOT NULL DEFAULT '';

ALTER TABLE billing_settings
  ADD COLUMN IF NOT EXISTS department_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS municipality_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS district_code VARCHAR(8) NOT NULL DEFAULT '';

-- DTE-specific invoice snapshots keep a fiscal document independent from
-- later edits to the customer or billing profile. Existing issued invoices are
-- backfilled once during this migration before any DTE can be prepared.
ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS customer_registration_number VARCHAR(40) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS customer_document_type_code VARCHAR(4) NOT NULL DEFAULT '36',
  ADD COLUMN IF NOT EXISTS customer_trade_name VARCHAR(240) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS customer_economic_activity VARCHAR(240) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS customer_economic_activity_code VARCHAR(12) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS customer_department_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS customer_municipality_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS customer_district_code VARCHAR(8) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS seller_department_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS seller_municipality_code VARCHAR(4) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS seller_district_code VARCHAR(8) NOT NULL DEFAULT '';

UPDATE invoices invoice
SET customer_registration_number = customer.tax_registration_number,
    customer_document_type_code = customer.document_type_code,
    customer_trade_name = customer.trade_name,
    customer_economic_activity = customer.economic_activity,
    customer_economic_activity_code = customer.economic_activity_code,
    customer_department_code = customer.department_code,
    customer_municipality_code = customer.municipality_code,
    customer_district_code = customer.district_code
FROM customers customer
WHERE customer.tenant_id = invoice.tenant_id
  AND customer.id = invoice.customer_id;

UPDATE invoices invoice
SET seller_department_code = settings.department_code,
    seller_municipality_code = settings.municipality_code,
    seller_district_code = settings.district_code
FROM billing_settings settings
WHERE settings.tenant_id = invoice.tenant_id;

ALTER TABLE invoice_items
  ADD COLUMN IF NOT EXISTS dte_item_type SMALLINT NOT NULL DEFAULT 2,
  ADD COLUMN IF NOT EXISTS dte_unit_code SMALLINT NOT NULL DEFAULT 59,
  ADD COLUMN IF NOT EXISTS dte_product_code VARCHAR(40) NOT NULL DEFAULT '';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'invoice_items_dte_item_type_chk'
  ) THEN
    ALTER TABLE invoice_items ADD CONSTRAINT invoice_items_dte_item_type_chk
      CHECK (dte_item_type IN (1, 2, 3, 4));
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'invoice_items_dte_unit_code_chk'
  ) THEN
    ALTER TABLE invoice_items ADD CONSTRAINT invoice_items_dte_unit_code_chk
      CHECK (dte_unit_code BETWEEN 1 AND 99);
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS dte_settings (
  tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  provider_mode VARCHAR(16) NOT NULL DEFAULT 'MOCK',
  environment VARCHAR(16) NOT NULL DEFAULT 'TEST',
  default_document_type VARCHAR(2) NOT NULL DEFAULT '01',
  schema_version INTEGER NOT NULL DEFAULT 1,
  establishment_type VARCHAR(4) NOT NULL DEFAULT '01',
  establishment_code VARCHAR(4) NOT NULL DEFAULT 'M001',
  point_of_sale_code VARCHAR(4) NOT NULL DEFAULT 'P001',
  auth_url TEXT NOT NULL DEFAULT '',
  signer_url TEXT NOT NULL DEFAULT '',
  reception_url TEXT NOT NULL DEFAULT '',
  invalidation_url TEXT NOT NULL DEFAULT '',
  query_url TEXT NOT NULL DEFAULT '',
  user_secret_ref TEXT NOT NULL DEFAULT '',
  password_secret_ref TEXT NOT NULL DEFAULT '',
  signing_password_secret_ref TEXT NOT NULL DEFAULT '',
  auto_submit_on_issue BOOLEAN NOT NULL DEFAULT FALSE,
  max_attempts INTEGER NOT NULL DEFAULT 5,
  retry_base_seconds INTEGER NOT NULL DEFAULT 60,
  next_control_number BIGINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT dte_settings_provider_chk CHECK (provider_mode IN ('MOCK', 'MH_HTTP')),
  CONSTRAINT dte_settings_environment_chk CHECK (environment IN ('TEST', 'PRODUCTION')),
  CONSTRAINT dte_settings_document_type_chk CHECK (default_document_type IN ('01', '03')),
  CONSTRAINT dte_settings_schema_chk CHECK (schema_version BETWEEN 1 AND 99),
  CONSTRAINT dte_settings_establishment_type_chk CHECK (establishment_type ~ '^[0-9A-Za-z]{1,4}$'),
  CONSTRAINT dte_settings_establishment_code_chk CHECK (establishment_code ~ '^[0-9A-Za-z]{4}$'),
  CONSTRAINT dte_settings_pos_code_chk CHECK (point_of_sale_code ~ '^[0-9A-Za-z]{4}$'),
  CONSTRAINT dte_settings_attempts_chk CHECK (max_attempts BETWEEN 1 AND 20),
  CONSTRAINT dte_settings_retry_chk CHECK (retry_base_seconds BETWEEN 5 AND 86400),
  CONSTRAINT dte_settings_sequence_chk CHECK (next_control_number >= 1)
);

CREATE TABLE IF NOT EXISTS dte_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  invoice_id UUID NOT NULL,
  document_type VARCHAR(2) NOT NULL,
  schema_version INTEGER NOT NULL DEFAULT 1,
  provider_mode VARCHAR(16) NOT NULL,
  environment VARCHAR(16) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'READY_TO_SIGN',
  generation_code UUID NOT NULL,
  control_number VARCHAR(40) NOT NULL,
  idempotency_key VARCHAR(64) NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  signed_document TEXT NOT NULL DEFAULT '',
  provider_request JSONB NOT NULL DEFAULT '{}'::jsonb,
  provider_response JSONB NOT NULL DEFAULT '{}'::jsonb,
  invalidation_request JSONB NOT NULL DEFAULT '{}'::jsonb,
  invalidation_response JSONB NOT NULL DEFAULT '{}'::jsonb,
  receipt_seal VARCHAR(160) NOT NULL DEFAULT '',
  provider_status VARCHAR(80) NOT NULL DEFAULT '',
  error_code VARCHAR(120) NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ,
  submitted_at TIMESTAMPTZ,
  accepted_at TIMESTAMPTZ,
  rejected_at TIMESTAMPTZ,
  invalidated_at TIMESTAMPTZ,
  invalidation_reason TEXT NOT NULL DEFAULT '',
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT dte_documents_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT dte_documents_invoice_fk FOREIGN KEY (tenant_id, invoice_id)
    REFERENCES invoices(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT dte_documents_type_chk CHECK (document_type IN ('01', '03')),
  CONSTRAINT dte_documents_provider_chk CHECK (provider_mode IN ('MOCK', 'MH_HTTP')),
  CONSTRAINT dte_documents_environment_chk CHECK (environment IN ('TEST', 'PRODUCTION')),
  CONSTRAINT dte_documents_status_chk CHECK (status IN (
    'READY_TO_SIGN', 'SUBMITTING', 'ACCEPTED', 'REJECTED',
    'RETRY_REQUIRED', 'INVALIDATION_PENDING', 'INVALIDATED', 'CANCELLED'
  )),
  CONSTRAINT dte_documents_attempt_chk CHECK (attempt_count >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS dte_documents_tenant_control_uniq
  ON dte_documents(tenant_id, control_number);
CREATE UNIQUE INDEX IF NOT EXISTS dte_documents_tenant_generation_uniq
  ON dte_documents(tenant_id, generation_code);
CREATE UNIQUE INDEX IF NOT EXISTS dte_documents_tenant_idempotency_uniq
  ON dte_documents(tenant_id, idempotency_key);
DROP INDEX IF EXISTS dte_documents_one_active_per_invoice;
CREATE UNIQUE INDEX dte_documents_one_active_per_invoice
  ON dte_documents(tenant_id, invoice_id)
  WHERE status NOT IN ('REJECTED', 'INVALIDATED', 'CANCELLED');
CREATE INDEX IF NOT EXISTS dte_documents_tenant_status_idx
  ON dte_documents(tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS dte_documents_retry_idx
  ON dte_documents(status, next_attempt_at)
  WHERE status = 'RETRY_REQUIRED';

CREATE TABLE IF NOT EXISTS dte_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  dte_document_id UUID NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT dte_events_document_fk FOREIGN KEY (tenant_id, dte_document_id)
    REFERENCES dte_documents(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS dte_events_document_idx
  ON dte_events(tenant_id, dte_document_id, created_at DESC);

INSERT INTO dte_settings (tenant_id)
SELECT tenant.id
FROM tenants tenant
ON CONFLICT (tenant_id) DO NOTHING;

-- Keep timestamps consistent with the existing trigger helper.
DROP TRIGGER IF EXISTS set_dte_settings_updated_at ON dte_settings;
CREATE TRIGGER set_dte_settings_updated_at
BEFORE UPDATE ON dte_settings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_dte_documents_updated_at ON dte_documents;
CREATE TRIGGER set_dte_documents_updated_at
BEFORE UPDATE ON dte_documents
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
