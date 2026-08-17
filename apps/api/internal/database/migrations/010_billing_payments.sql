-- RentStage v0.11.0 — Billing & Payments Core.
-- Internal invoices and payment records are operational/accounting documents.
-- They are intentionally separate from El Salvador DTE transmission, which is
-- reserved for a dedicated fiscal provider in a later release.

ALTER TABLE customers
  ADD COLUMN IF NOT EXISTS tax_id VARCHAR(40) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS tax_registration_number VARCHAR(40) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS billing_address TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS billing_settings (
  tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  legal_name VARCHAR(240) NOT NULL DEFAULT '',
  trade_name VARCHAR(240) NOT NULL DEFAULT '',
  tax_id VARCHAR(40) NOT NULL DEFAULT '',
  tax_registration_number VARCHAR(40) NOT NULL DEFAULT '',
  economic_activity VARCHAR(240) NOT NULL DEFAULT '',
  economic_activity_code VARCHAR(40) NOT NULL DEFAULT '',
  fiscal_address TEXT NOT NULL DEFAULT '',
  department VARCHAR(120) NOT NULL DEFAULT '',
  municipality VARCHAR(120) NOT NULL DEFAULT '',
  district VARCHAR(120) NOT NULL DEFAULT '',
  email VARCHAR(320) NOT NULL DEFAULT '',
  phone VARCHAR(80) NOT NULL DEFAULT '',
  prices_include_tax BOOLEAN NOT NULL DEFAULT FALSE,
  default_tax_rate NUMERIC(5,2) NOT NULL DEFAULT 13.00,
  default_payment_terms_days INTEGER NOT NULL DEFAULT 0,
  invoice_prefix VARCHAR(12) NOT NULL DEFAULT 'INV',
  next_invoice_number BIGINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT billing_settings_tax_rate_chk CHECK (default_tax_rate BETWEEN 0 AND 100),
  CONSTRAINT billing_settings_terms_chk CHECK (default_payment_terms_days BETWEEN 0 AND 365),
  CONSTRAINT billing_settings_prefix_chk CHECK (invoice_prefix ~ '^[A-Za-z0-9_-]{1,12}$'),
  CONSTRAINT billing_settings_next_number_chk CHECK (next_invoice_number >= 1)
);

CREATE TABLE IF NOT EXISTS tax_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  code VARCHAR(30) NOT NULL,
  name VARCHAR(160) NOT NULL,
  category VARCHAR(24) NOT NULL,
  rate NUMERIC(5,2) NOT NULL DEFAULT 0,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
  valid_until DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT tax_rules_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT tax_rules_tenant_code_uniq UNIQUE (tenant_id, code),
  CONSTRAINT tax_rules_category_chk CHECK (category IN ('TAXABLE', 'EXEMPT', 'NON_TAXABLE')),
  CONSTRAINT tax_rules_rate_chk CHECK (rate BETWEEN 0 AND 100),
  CONSTRAINT tax_rules_zero_rate_chk CHECK (category = 'TAXABLE' OR rate = 0),
  CONSTRAINT tax_rules_validity_chk CHECK (valid_until IS NULL OR valid_until >= valid_from)
);

CREATE UNIQUE INDEX IF NOT EXISTS tax_rules_one_default_per_tenant
  ON tax_rules(tenant_id)
  WHERE is_default = TRUE AND active = TRUE;

CREATE TABLE IF NOT EXISTS invoices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  customer_id UUID NOT NULL,
  quote_id UUID,
  reservation_id UUID,
  source_type VARCHAR(20) NOT NULL DEFAULT 'MANUAL',
  invoice_number BIGINT,
  invoice_prefix VARCHAR(12) NOT NULL DEFAULT 'INV',
  status VARCHAR(24) NOT NULL DEFAULT 'DRAFT',
  issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
  due_date DATE NOT NULL DEFAULT CURRENT_DATE,
  currency CHAR(3) NOT NULL DEFAULT 'USD',
  prices_include_tax BOOLEAN NOT NULL DEFAULT FALSE,
  customer_name VARCHAR(240) NOT NULL,
  customer_tax_id VARCHAR(40) NOT NULL DEFAULT '',
  customer_email VARCHAR(320) NOT NULL DEFAULT '',
  customer_phone VARCHAR(80) NOT NULL DEFAULT '',
  customer_address TEXT NOT NULL DEFAULT '',
  seller_legal_name VARCHAR(240) NOT NULL DEFAULT '',
  seller_trade_name VARCHAR(240) NOT NULL DEFAULT '',
  seller_tax_id VARCHAR(40) NOT NULL DEFAULT '',
  seller_registration_number VARCHAR(40) NOT NULL DEFAULT '',
  seller_economic_activity VARCHAR(240) NOT NULL DEFAULT '',
  seller_economic_activity_code VARCHAR(40) NOT NULL DEFAULT '',
  seller_address TEXT NOT NULL DEFAULT '',
  seller_email VARCHAR(320) NOT NULL DEFAULT '',
  seller_phone VARCHAR(80) NOT NULL DEFAULT '',
  taxable_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  exempt_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  non_taxable_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  tax_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  total_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  paid_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  balance_due NUMERIC(14,2) GENERATED ALWAYS AS (total_amount - paid_amount) STORED,
  fiscal_status VARCHAR(24) NOT NULL DEFAULT 'NOT_READY',
  notes TEXT NOT NULL DEFAULT '',
  terms TEXT NOT NULL DEFAULT '',
  issued_at TIMESTAMPTZ,
  issued_by UUID REFERENCES users(id) ON DELETE SET NULL,
  voided_at TIMESTAMPTZ,
  voided_by UUID REFERENCES users(id) ON DELETE SET NULL,
  void_reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT invoices_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT invoices_customer_fk FOREIGN KEY (tenant_id, customer_id)
    REFERENCES customers(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT invoices_quote_fk FOREIGN KEY (tenant_id, quote_id)
    REFERENCES quotes(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT invoices_reservation_fk FOREIGN KEY (tenant_id, reservation_id)
    REFERENCES reservations(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT invoices_source_chk CHECK (source_type IN ('MANUAL', 'QUOTE', 'RESERVATION')),
  CONSTRAINT invoices_source_consistent_chk CHECK (
    (source_type = 'MANUAL' AND quote_id IS NULL AND reservation_id IS NULL)
    OR (source_type = 'QUOTE' AND quote_id IS NOT NULL)
    OR (source_type = 'RESERVATION' AND reservation_id IS NOT NULL)
  ),
  CONSTRAINT invoices_status_chk CHECK (status IN ('DRAFT', 'ISSUED', 'PARTIALLY_PAID', 'PAID', 'VOID')),
  CONSTRAINT invoices_dates_chk CHECK (due_date >= issue_date),
  CONSTRAINT invoices_number_chk CHECK (invoice_number IS NULL OR invoice_number >= 1),
  CONSTRAINT invoices_amounts_chk CHECK (
    taxable_amount >= 0 AND exempt_amount >= 0 AND non_taxable_amount >= 0
    AND tax_amount >= 0 AND total_amount >= 0 AND paid_amount >= 0
    AND paid_amount <= total_amount
  ),
  CONSTRAINT invoices_total_chk CHECK (
    total_amount = taxable_amount + exempt_amount + non_taxable_amount + tax_amount
  ),
  CONSTRAINT invoices_issued_number_chk CHECK (
    (status = 'DRAFT' AND invoice_number IS NULL AND issued_at IS NULL)
    OR (status = 'VOID' AND (invoice_number IS NULL OR issued_at IS NOT NULL))
    OR (status IN ('ISSUED', 'PARTIALLY_PAID', 'PAID') AND invoice_number IS NOT NULL AND issued_at IS NOT NULL)
  ),
  CONSTRAINT invoices_fiscal_status_chk CHECK (fiscal_status IN (
    'NOT_READY', 'READY_FOR_DTE', 'SUBMITTED', 'ACCEPTED', 'REJECTED', 'VOIDED'
  ))
);

CREATE UNIQUE INDEX IF NOT EXISTS invoices_tenant_number_uniq
  ON invoices(tenant_id, invoice_number)
  WHERE invoice_number IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS invoices_active_quote_uniq
  ON invoices(tenant_id, quote_id)
  WHERE quote_id IS NOT NULL AND status <> 'VOID';
CREATE UNIQUE INDEX IF NOT EXISTS invoices_active_reservation_uniq
  ON invoices(tenant_id, reservation_id)
  WHERE reservation_id IS NOT NULL AND status <> 'VOID';
CREATE INDEX IF NOT EXISTS invoices_tenant_status_due_idx
  ON invoices(tenant_id, status, due_date, created_at DESC);
CREATE INDEX IF NOT EXISTS invoices_tenant_customer_idx
  ON invoices(tenant_id, customer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS invoice_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  invoice_id UUID NOT NULL,
  resource_id UUID,
  tax_rule_id UUID,
  description VARCHAR(500) NOT NULL,
  quantity NUMERIC(12,3) NOT NULL,
  unit_price NUMERIC(14,4) NOT NULL,
  discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  gross_amount NUMERIC(14,2) NOT NULL,
  net_amount NUMERIC(14,2) NOT NULL,
  tax_code VARCHAR(30) NOT NULL,
  tax_category VARCHAR(24) NOT NULL,
  tax_rate NUMERIC(5,2) NOT NULL,
  tax_amount NUMERIC(14,2) NOT NULL,
  line_total NUMERIC(14,2) NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT invoice_items_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT invoice_items_invoice_fk FOREIGN KEY (tenant_id, invoice_id)
    REFERENCES invoices(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT invoice_items_resource_fk FOREIGN KEY (tenant_id, resource_id)
    REFERENCES resources(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT invoice_items_tax_rule_fk FOREIGN KEY (tenant_id, tax_rule_id)
    REFERENCES tax_rules(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT invoice_items_quantity_chk CHECK (quantity > 0 AND quantity <= 100000),
  CONSTRAINT invoice_items_unit_price_chk CHECK (unit_price >= 0),
  CONSTRAINT invoice_items_amounts_chk CHECK (
    discount_amount >= 0 AND gross_amount >= 0 AND net_amount >= 0
    AND tax_amount >= 0 AND line_total >= 0 AND discount_amount <= gross_amount
  ),
  CONSTRAINT invoice_items_total_chk CHECK (line_total = net_amount + tax_amount),
  CONSTRAINT invoice_items_category_chk CHECK (tax_category IN ('TAXABLE', 'EXEMPT', 'NON_TAXABLE')),
  CONSTRAINT invoice_items_tax_rate_chk CHECK (tax_rate BETWEEN 0 AND 100),
  CONSTRAINT invoice_items_zero_tax_chk CHECK (tax_category = 'TAXABLE' OR tax_rate = 0),
  CONSTRAINT invoice_items_sort_chk CHECK (sort_order >= 0)
);

CREATE INDEX IF NOT EXISTS invoice_items_invoice_idx
  ON invoice_items(tenant_id, invoice_id, sort_order, id);

CREATE TABLE IF NOT EXISTS invoice_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  invoice_id UUID NOT NULL,
  event_type VARCHAR(40) NOT NULL,
  actor_id VARCHAR(180) NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT invoice_events_invoice_fk FOREIGN KEY (tenant_id, invoice_id)
    REFERENCES invoices(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT invoice_events_type_chk CHECK (event_type IN (
    'CREATED', 'UPDATED', 'ISSUED', 'PAYMENT_APPLIED', 'PAYMENT_VOIDED', 'VOIDED'
  ))
);

CREATE INDEX IF NOT EXISTS invoice_events_invoice_idx
  ON invoice_events(tenant_id, invoice_id, created_at DESC);

CREATE TABLE IF NOT EXISTS payments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  customer_id UUID NOT NULL,
  payment_number BIGINT GENERATED BY DEFAULT AS IDENTITY,
  status VARCHAR(20) NOT NULL DEFAULT 'CONFIRMED',
  amount NUMERIC(14,2) NOT NULL,
  currency CHAR(3) NOT NULL DEFAULT 'USD',
  method VARCHAR(24) NOT NULL,
  reference VARCHAR(240) NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  received_at TIMESTAMPTZ NOT NULL,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  voided_at TIMESTAMPTZ,
  voided_by UUID REFERENCES users(id) ON DELETE SET NULL,
  void_reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT payments_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT payments_tenant_number_uniq UNIQUE (tenant_id, payment_number),
  CONSTRAINT payments_customer_fk FOREIGN KEY (tenant_id, customer_id)
    REFERENCES customers(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT payments_status_chk CHECK (status IN ('CONFIRMED', 'VOIDED')),
  CONSTRAINT payments_amount_chk CHECK (amount > 0),
  CONSTRAINT payments_method_chk CHECK (method IN ('CASH', 'BANK_TRANSFER', 'CARD', 'CHECK', 'OTHER'))
);

CREATE INDEX IF NOT EXISTS payments_tenant_received_idx
  ON payments(tenant_id, received_at DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS payments_tenant_customer_idx
  ON payments(tenant_id, customer_id, received_at DESC);

CREATE TABLE IF NOT EXISTS payment_allocations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  payment_id UUID NOT NULL,
  invoice_id UUID NOT NULL,
  amount NUMERIC(14,2) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT payment_allocations_payment_fk FOREIGN KEY (tenant_id, payment_id)
    REFERENCES payments(tenant_id, id) ON DELETE CASCADE,
  CONSTRAINT payment_allocations_invoice_fk FOREIGN KEY (tenant_id, invoice_id)
    REFERENCES invoices(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT payment_allocations_payment_invoice_uniq UNIQUE (tenant_id, payment_id, invoice_id),
  CONSTRAINT payment_allocations_amount_chk CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS payment_allocations_invoice_idx
  ON payment_allocations(tenant_id, invoice_id, created_at DESC);

CREATE TABLE IF NOT EXISTS security_deposits (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  reservation_id UUID NOT NULL,
  customer_id UUID NOT NULL,
  deposit_number BIGINT GENERATED BY DEFAULT AS IDENTITY,
  status VARCHAR(24) NOT NULL DEFAULT 'PENDING',
  amount NUMERIC(14,2) NOT NULL,
  returned_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  retained_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  balance_amount NUMERIC(14,2) GENERATED ALWAYS AS (amount - returned_amount - retained_amount) STORED,
  currency CHAR(3) NOT NULL DEFAULT 'USD',
  method VARCHAR(24) NOT NULL DEFAULT 'OTHER',
  reference VARCHAR(240) NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  received_at TIMESTAMPTZ,
  settled_at TIMESTAMPTZ,
  settlement_reason TEXT NOT NULL DEFAULT '',
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT security_deposits_tenant_id_uniq UNIQUE (tenant_id, id),
  CONSTRAINT security_deposits_tenant_number_uniq UNIQUE (tenant_id, deposit_number),
  CONSTRAINT security_deposits_reservation_fk FOREIGN KEY (tenant_id, reservation_id)
    REFERENCES reservations(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT security_deposits_customer_fk FOREIGN KEY (tenant_id, customer_id)
    REFERENCES customers(tenant_id, id) ON DELETE RESTRICT,
  CONSTRAINT security_deposits_status_chk CHECK (status IN (
    'PENDING', 'RECEIVED', 'PARTIALLY_SETTLED', 'RETURNED', 'RETAINED', 'SETTLED'
  )),
  CONSTRAINT security_deposits_amount_chk CHECK (
    amount > 0 AND returned_amount >= 0 AND retained_amount >= 0
    AND returned_amount + retained_amount <= amount
  ),
  CONSTRAINT security_deposits_method_chk CHECK (method IN ('CASH', 'BANK_TRANSFER', 'CARD', 'CHECK', 'OTHER')),
  CONSTRAINT security_deposits_received_chk CHECK (
    (status = 'PENDING' AND received_at IS NULL)
    OR (status <> 'PENDING' AND received_at IS NOT NULL)
  )
);

CREATE INDEX IF NOT EXISTS security_deposits_tenant_status_idx
  ON security_deposits(tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS security_deposits_reservation_idx
  ON security_deposits(tenant_id, reservation_id, created_at DESC);

DROP TRIGGER IF EXISTS billing_settings_set_updated_at ON billing_settings;
CREATE TRIGGER billing_settings_set_updated_at
BEFORE UPDATE ON billing_settings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS tax_rules_set_updated_at ON tax_rules;
CREATE TRIGGER tax_rules_set_updated_at
BEFORE UPDATE ON tax_rules
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS invoices_set_updated_at ON invoices;
CREATE TRIGGER invoices_set_updated_at
BEFORE UPDATE ON invoices
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS payments_set_updated_at ON payments;
CREATE TRIGGER payments_set_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS security_deposits_set_updated_at ON security_deposits;
CREATE TRIGGER security_deposits_set_updated_at
BEFORE UPDATE ON security_deposits
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO billing_settings (
  tenant_id, legal_name, trade_name, fiscal_address, email, phone
)
SELECT
  tenant.id,
  COALESCE(tenant.legal_name, tenant.name, ''),
  tenant.name,
  COALESCE(tenant.address, ''),
  COALESCE(tenant.email, ''),
  COALESCE(tenant.phone, '')
FROM tenants tenant
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO tax_rules (tenant_id, code, name, category, rate, active, is_default)
SELECT tenant.id, 'IVA', 'IVA estándar', 'TAXABLE', 13.00, TRUE, TRUE
FROM tenants tenant
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO tax_rules (tenant_id, code, name, category, rate, active, is_default)
SELECT tenant.id, 'EXEMPT', 'Venta exenta', 'EXEMPT', 0.00, TRUE, FALSE
FROM tenants tenant
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO tax_rules (tenant_id, code, name, category, rate, active, is_default)
SELECT tenant.id, 'NON_TAXABLE', 'Venta no sujeta', 'NON_TAXABLE', 0.00, TRUE, FALSE
FROM tenants tenant
ON CONFLICT (tenant_id, code) DO NOTHING;
