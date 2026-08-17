ALTER TABLE tenants
  ADD COLUMN IF NOT EXISTS logo_url TEXT,
  ADD COLUMN IF NOT EXISTS address TEXT;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS identity_uid TEXT,
  ADD COLUMN IF NOT EXISTS avatar_url TEXT,
  ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_identity_uid
  ON users(identity_uid)
  WHERE identity_uid IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower
  ON users(LOWER(email));

ALTER TABLE tenant_memberships
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ACTIVE',
  ADD COLUMN IF NOT EXISTS invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS joined_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE tenant_memberships
SET joined_at = COALESCE(joined_at, created_at),
    status = COALESCE(NULLIF(status, ''), 'ACTIVE');

ALTER TABLE tenant_memberships
  DROP CONSTRAINT IF EXISTS tenant_memberships_status_check;
ALTER TABLE tenant_memberships
  ADD CONSTRAINT tenant_memberships_status_check
  CHECK (status IN ('INVITED', 'ACTIVE', 'SUSPENDED', 'REMOVED'));

CREATE TABLE IF NOT EXISTS tenant_invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('ADMIN', 'MANAGER', 'STAFF')),
  token_hash TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACCEPTED', 'REVOKED', 'EXPIRED')),
  expires_at TIMESTAMPTZ NOT NULL,
  invited_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  accepted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  accepted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  active_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
  locale TEXT NOT NULL DEFAULT 'es' CHECK (locale IN ('es', 'en')),
  timezone TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memberships_user_status
  ON tenant_memberships(user_id, status);
CREATE INDEX IF NOT EXISTS idx_memberships_tenant_status
  ON tenant_memberships(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_invitations_tenant_status
  ON tenant_invitations(tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invitations_email_status
  ON tenant_invitations(LOWER(email), status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_tenant_email_pending
  ON tenant_invitations(tenant_id, LOWER(email))
  WHERE status = 'PENDING';

DROP TRIGGER IF EXISTS tenant_memberships_set_updated_at ON tenant_memberships;
CREATE TRIGGER tenant_memberships_set_updated_at
BEFORE UPDATE ON tenant_memberships
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS tenant_invitations_set_updated_at ON tenant_invitations;
CREATE TRIGGER tenant_invitations_set_updated_at
BEFORE UPDATE ON tenant_invitations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS user_preferences_set_updated_at ON user_preferences;
CREATE TRIGGER user_preferences_set_updated_at
BEFORE UPDATE ON user_preferences
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
