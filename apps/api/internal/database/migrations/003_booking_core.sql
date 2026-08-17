-- Booking Core: temporal availability, reservations, status history, and quote conversion.

ALTER TABLE reservations RENAME COLUMN start_at TO block_start_at;
ALTER TABLE reservations RENAME COLUMN end_at TO block_end_at;

ALTER TABLE reservations
  ADD COLUMN event_start_at TIMESTAMPTZ,
  ADD COLUMN event_end_at TIMESTAMPTZ,
  ADD COLUMN subtotal NUMERIC(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN extra_charges NUMERIC(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN total NUMERIC(12,2) NOT NULL DEFAULT 0;

UPDATE reservations
SET event_start_at = block_start_at,
    event_end_at = block_end_at
WHERE event_start_at IS NULL OR event_end_at IS NULL;

ALTER TABLE reservations
  ALTER COLUMN event_start_at SET NOT NULL,
  ALTER COLUMN event_end_at SET NOT NULL;

ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_check;
ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_status_check;

ALTER TABLE reservations
  ADD CONSTRAINT reservations_block_period_valid
    CHECK (block_end_at > block_start_at),
  ADD CONSTRAINT reservations_event_period_valid
    CHECK (event_end_at > event_start_at),
  ADD CONSTRAINT reservations_event_within_block
    CHECK (event_start_at >= block_start_at AND event_end_at <= block_end_at),
  ADD CONSTRAINT reservations_status_check
    CHECK (status IN (
      'PENDING', 'CONFIRMED', 'PREPARING', 'READY',
      'CHECKED_OUT', 'RETURNED', 'COMPLETED', 'CANCELLED'
    )),
  ADD CONSTRAINT reservations_subtotal_nonnegative CHECK (subtotal >= 0),
  ADD CONSTRAINT reservations_discount_nonnegative CHECK (discount_amount >= 0),
  ADD CONSTRAINT reservations_extra_charges_nonnegative CHECK (extra_charges >= 0),
  ADD CONSTRAINT reservations_total_nonnegative CHECK (total >= 0),
  ADD CONSTRAINT reservations_discount_within_subtotal CHECK (discount_amount <= subtotal),
  ADD CONSTRAINT reservations_total_consistent
    CHECK (total = subtotal - discount_amount + extra_charges);

ALTER TABLE reservation_items
  ADD COLUMN quote_item_id UUID,
  ADD COLUMN description TEXT NOT NULL DEFAULT '',
  ADD COLUMN discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN line_total NUMERIC(12,2) NOT NULL DEFAULT 0;

UPDATE reservation_items ri
SET description = r.name,
    line_total = ri.quantity * ri.unit_price
FROM resources r
WHERE r.tenant_id = ri.tenant_id
  AND r.id = ri.resource_id
  AND ri.description = '';

ALTER TABLE reservation_items
  ADD CONSTRAINT reservation_items_quote_item_fk
    FOREIGN KEY (tenant_id, quote_item_id)
    REFERENCES quote_items(tenant_id, id)
    ON DELETE RESTRICT,
  ADD CONSTRAINT reservation_items_discount_nonnegative CHECK (discount_amount >= 0),
  ADD CONSTRAINT reservation_items_discount_within_gross
    CHECK (discount_amount <= quantity * unit_price),
  ADD CONSTRAINT reservation_items_line_total_consistent
    CHECK (line_total = quantity * unit_price - discount_amount);

CREATE TABLE reservation_status_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  reservation_id UUID NOT NULL,
  from_status TEXT,
  to_status TEXT NOT NULL CHECK (to_status IN (
    'PENDING', 'CONFIRMED', 'PREPARING', 'READY',
    'CHECKED_OUT', 'RETURNED', 'COMPLETED', 'CANCELLED'
  )),
  actor_id TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, reservation_id)
    REFERENCES reservations(tenant_id, id)
    ON DELETE CASCADE
);

DROP INDEX IF EXISTS idx_reservations_tenant_period;
CREATE INDEX idx_reservations_tenant_block_period
  ON reservations(tenant_id, block_start_at, block_end_at);
CREATE INDEX idx_reservations_tenant_status_start
  ON reservations(tenant_id, status, block_start_at);
CREATE INDEX idx_reservations_tenant_customer_created
  ON reservations(tenant_id, customer_id, created_at DESC);
CREATE UNIQUE INDEX idx_reservations_tenant_quote_unique
  ON reservations(tenant_id, quote_id)
  WHERE quote_id IS NOT NULL;
CREATE INDEX idx_reservation_status_history_reservation
  ON reservation_status_history(tenant_id, reservation_id, created_at);
