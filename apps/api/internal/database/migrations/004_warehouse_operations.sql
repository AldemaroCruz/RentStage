-- Warehouse Operations: physical asset assignment, checkout, return inspection, and activity history.

ALTER TABLE reservations
  ADD COLUMN checked_out_at TIMESTAMPTZ,
  ADD COLUMN checked_out_by TEXT,
  ADD COLUMN checkout_notes TEXT NOT NULL DEFAULT '',
  ADD COLUMN returned_at TIMESTAMPTZ,
  ADD COLUMN returned_by TEXT,
  ADD COLUMN return_notes TEXT NOT NULL DEFAULT '';

ALTER TABLE reservation_assets
  ADD COLUMN id UUID DEFAULT gen_random_uuid(),
  ADD COLUMN reservation_id UUID,
  ADD COLUMN assigned_by TEXT NOT NULL DEFAULT 'system',
  ADD COLUMN checked_out_at TIMESTAMPTZ,
  ADD COLUMN checked_out_by TEXT,
  ADD COLUMN checkout_notes TEXT NOT NULL DEFAULT '',
  ADD COLUMN returned_at TIMESTAMPTZ,
  ADD COLUMN returned_by TEXT,
  ADD COLUMN return_condition TEXT,
  ADD COLUMN return_notes TEXT NOT NULL DEFAULT '',
  ADD COLUMN released_at TIMESTAMPTZ,
  ADD COLUMN released_by TEXT,
  ADD COLUMN release_reason TEXT NOT NULL DEFAULT '';

UPDATE reservation_assets ra
SET reservation_id = ri.reservation_id,
    id = COALESCE(ra.id, gen_random_uuid())
FROM reservation_items ri
WHERE ri.tenant_id = ra.tenant_id
  AND ri.id = ra.reservation_item_id
  AND (ra.reservation_id IS NULL OR ra.id IS NULL);

ALTER TABLE reservation_assets
  ALTER COLUMN id SET NOT NULL,
  ALTER COLUMN reservation_id SET NOT NULL;

ALTER TABLE reservation_assets
  DROP CONSTRAINT reservation_assets_pkey;

ALTER TABLE reservation_assets
  ADD CONSTRAINT reservation_assets_pkey PRIMARY KEY (id),
  ADD CONSTRAINT reservation_assets_reservation_fk
    FOREIGN KEY (tenant_id, reservation_id)
    REFERENCES reservations(tenant_id, id)
    ON DELETE CASCADE,
  ADD CONSTRAINT reservation_assets_return_condition_check
    CHECK (return_condition IS NULL OR return_condition IN (
      'GOOD', 'MAINTENANCE_REQUIRED', 'DAMAGED', 'LOST'
    )),
  ADD CONSTRAINT reservation_assets_return_fields_consistent
    CHECK (
      (returned_at IS NULL AND returned_by IS NULL AND return_condition IS NULL)
      OR
      (returned_at IS NOT NULL AND returned_by IS NOT NULL AND return_condition IS NOT NULL)
    ),
  ADD CONSTRAINT reservation_assets_release_fields_consistent
    CHECK (
      (released_at IS NULL AND released_by IS NULL)
      OR
      (released_at IS NOT NULL AND released_by IS NOT NULL)
    );

CREATE UNIQUE INDEX idx_reservation_assets_active_item_asset
  ON reservation_assets(tenant_id, reservation_item_id, asset_id)
  WHERE released_at IS NULL;

CREATE UNIQUE INDEX idx_reservation_assets_active_reservation_asset
  ON reservation_assets(tenant_id, reservation_id, asset_id)
  WHERE released_at IS NULL;

CREATE INDEX idx_reservation_assets_reservation_active
  ON reservation_assets(tenant_id, reservation_id, reservation_item_id)
  WHERE released_at IS NULL;

CREATE INDEX idx_reservation_assets_asset_active
  ON reservation_assets(tenant_id, asset_id)
  WHERE released_at IS NULL;

CREATE TABLE reservation_activity_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  reservation_id UUID NOT NULL,
  event_type TEXT NOT NULL CHECK (event_type IN (
    'ASSET_ASSIGNED',
    'ASSET_UNASSIGNED',
    'ASSET_CHECKED_OUT',
    'ASSET_RETURNED',
    'ASSIGNMENTS_RELEASED'
  )),
  asset_id UUID,
  actor_id TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, reservation_id)
    REFERENCES reservations(tenant_id, id)
    ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, asset_id)
    REFERENCES assets(tenant_id, id)
    ON DELETE RESTRICT
);

CREATE INDEX idx_reservation_activity_reservation
  ON reservation_activity_events(tenant_id, reservation_id, created_at, id);

CREATE INDEX idx_reservation_activity_asset
  ON reservation_activity_events(tenant_id, asset_id, created_at DESC)
  WHERE asset_id IS NOT NULL;
