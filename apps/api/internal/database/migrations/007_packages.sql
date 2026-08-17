-- RentStage v0.8 — reusable tenant-scoped commercial packages.

CREATE TABLE packages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  guest_capacity INTEGER,
  pricing_mode TEXT NOT NULL DEFAULT 'SUM_ITEMS'
    CHECK (pricing_mode IN ('SUM_ITEMS', 'FIXED')),
  fixed_price NUMERIC(12,2),
  image_url TEXT,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, slug),
  CONSTRAINT packages_name_not_blank CHECK (BTRIM(name) <> ''),
  CONSTRAINT packages_name_length CHECK (CHAR_LENGTH(name) <= 180),
  CONSTRAINT packages_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
  CONSTRAINT packages_slug_length CHECK (CHAR_LENGTH(slug) <= 140),
  CONSTRAINT packages_description_length CHECK (CHAR_LENGTH(description) <= 4000),
  CONSTRAINT packages_image_url_length CHECK (image_url IS NULL OR CHAR_LENGTH(image_url) <= 2000),
  CONSTRAINT packages_guest_capacity_positive
    CHECK (guest_capacity IS NULL OR (guest_capacity > 0 AND guest_capacity <= 1000000)),
  CONSTRAINT packages_fixed_price_nonnegative
    CHECK (fixed_price IS NULL OR fixed_price >= 0),
  CONSTRAINT packages_pricing_consistent CHECK (
    (pricing_mode = 'SUM_ITEMS' AND fixed_price IS NULL)
    OR
    (pricing_mode = 'FIXED' AND fixed_price IS NOT NULL)
  )
);

CREATE TABLE package_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  package_id UUID NOT NULL,
  resource_id UUID NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  quantity INTEGER NOT NULL CHECK (quantity > 0 AND quantity <= 10000),
  unit_price_override NUMERIC(12,2),
  sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, package_id, resource_id),
  CONSTRAINT package_items_description_length CHECK (CHAR_LENGTH(description) <= 500),
  CONSTRAINT package_items_price_override_nonnegative
    CHECK (unit_price_override IS NULL OR unit_price_override >= 0),
  FOREIGN KEY (tenant_id, package_id)
    REFERENCES packages(tenant_id, id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, resource_id)
    REFERENCES resources(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_packages_tenant_active_updated
  ON packages(tenant_id, active, updated_at DESC);

CREATE INDEX idx_package_items_package_sort
  ON package_items(tenant_id, package_id, sort_order, id);

CREATE INDEX idx_package_items_resource
  ON package_items(tenant_id, resource_id);

DROP TRIGGER IF EXISTS packages_set_updated_at ON packages;
CREATE TRIGGER packages_set_updated_at
BEFORE UPDATE ON packages
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS package_items_set_updated_at ON package_items;
CREATE TRIGGER package_items_set_updated_at
BEFORE UPDATE ON package_items
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
