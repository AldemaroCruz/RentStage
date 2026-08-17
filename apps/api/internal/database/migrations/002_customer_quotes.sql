-- Customer and quote workflow indexes and monetary safeguards.

CREATE INDEX IF NOT EXISTS idx_customers_tenant_source
  ON customers(tenant_id, source);

CREATE INDEX IF NOT EXISTS idx_customers_tenant_phone
  ON customers(tenant_id, phone)
  WHERE phone IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_customers_tenant_email
  ON customers(tenant_id, LOWER(email))
  WHERE email IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_quotes_tenant_customer_created
  ON quotes(tenant_id, customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_quote_items_tenant_quote
  ON quote_items(tenant_id, quote_id);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quotes_subtotal_nonnegative'
      AND conrelid = 'quotes'::regclass
  ) THEN
    ALTER TABLE quotes
      ADD CONSTRAINT quotes_subtotal_nonnegative CHECK (subtotal >= 0);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quotes_discount_nonnegative'
      AND conrelid = 'quotes'::regclass
  ) THEN
    ALTER TABLE quotes
      ADD CONSTRAINT quotes_discount_nonnegative CHECK (discount_amount >= 0);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quotes_extra_charges_nonnegative'
      AND conrelid = 'quotes'::regclass
  ) THEN
    ALTER TABLE quotes
      ADD CONSTRAINT quotes_extra_charges_nonnegative CHECK (extra_charges >= 0);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quotes_total_nonnegative'
      AND conrelid = 'quotes'::regclass
  ) THEN
    ALTER TABLE quotes
      ADD CONSTRAINT quotes_total_nonnegative CHECK (total >= 0);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quotes_discount_within_subtotal'
      AND conrelid = 'quotes'::regclass
  ) THEN
    ALTER TABLE quotes
      ADD CONSTRAINT quotes_discount_within_subtotal CHECK (discount_amount <= subtotal);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quotes_total_consistent'
      AND conrelid = 'quotes'::regclass
  ) THEN
    ALTER TABLE quotes
      ADD CONSTRAINT quotes_total_consistent CHECK (total = subtotal - discount_amount + extra_charges);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quote_items_discount_within_gross'
      AND conrelid = 'quote_items'::regclass
  ) THEN
    ALTER TABLE quote_items
      ADD CONSTRAINT quote_items_discount_within_gross CHECK (discount_amount <= quantity * unit_price);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'quote_items_line_total_consistent'
      AND conrelid = 'quote_items'::regclass
  ) THEN
    ALTER TABLE quote_items
      ADD CONSTRAINT quote_items_line_total_consistent CHECK (line_total = quantity * unit_price - discount_amount);
  END IF;
END
$$;
