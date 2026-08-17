# Upgrade RentStage v0.5.0 → v0.6.0

This package upgrades an existing v0.5 installation to the first authenticated, role-aware, multi-workspace RentStage release without replacing its PostgreSQL volume.

## What is preserved

- `.env`, including `POSTGRES_PORT=5433` on Windows.
- Tenant and all organization configuration already present.
- Categories, resources, and physical assets.
- Customers and quotes.
- Reservations, prices, status history, schedule history, and source attribution.
- Warehouse assignments, check-out records, return inspections, and physical activity history.
- Calendar and operations data.
- Existing audit events.

## What changes

- Every application page except login, signup, and invitation preview now requires an authenticated session.
- Every business API route now resolves a tenant from an active membership.
- Previous local development headers are no longer used as authority.
- Audit mutations use the authenticated RentStage user.
- Docker Compose adds a Firebase Authentication emulator service.
- The web image adds Firebase JS SDK support.
- The Go API adds Firebase Admin SDK support.

## New migration

```text
006_identity_access.sql
```

It adds:

```text
tenants.logo_url
tenants.address

users.identity_uid
users.avatar_url
users.email_verified
users.last_login_at

tenant_memberships.status
tenant_memberships.invited_by
tenant_memberships.joined_at
tenant_memberships.updated_at

tenant_invitations
user_preferences
```

The existing AudioPro demo owner is linked by normalized email when the API bootstraps the local Authentication emulator.

## 1. Back up PostgreSQL

From PowerShell:

```powershell
cd C:\Users\itres\Documents\rentstage-starter
New-Item -ItemType Directory -Force .\backups | Out-Null

docker compose exec db `
  pg_dump `
  -U rentstage `
  -d rentstage `
  -Fc `
  -f /tmp/rentstage-v0.5.0-before-v0.6.dump

docker compose cp `
  db:/tmp/rentstage-v0.5.0-before-v0.6.dump `
  .\backups\rentstage-v0.5.0-before-v0.6.dump

Get-Item .\backups\rentstage-v0.5.0-before-v0.6.dump
```

## 2. Stop containers without deleting volumes

```powershell
docker compose down
```

Do **not** run:

```powershell
docker compose down -v
```

`-v` would delete PostgreSQL and, after the upgrade, the Authentication emulator's local export volume.

## 3. Extract the upgrade

Extract:

```text
rentstage-upgrade-v0.6.0.zip
```

directly into:

```text
C:\Users\itres\Documents\rentstage-starter
```

Allow replacement of existing files. The upgrade ZIP does not contain `.env`, database backups, or generated build directories.

## 4. Merge new environment settings

Your existing `.env` is preserved, so compare it with the new `.env.example` and add these settings if they are missing:

```env
AUTH_EMULATOR_PORT=9099
AUTH_EMULATOR_UI_PORT=4000

FIREBASE_PROJECT_ID=demo-rentstage
FIREBASE_API_KEY=rentstage-local-api-key
FIREBASE_AUTH_DOMAIN=demo-rentstage.firebaseapp.com

LOCAL_AUTH_BOOTSTRAP=true
LOCAL_OWNER_EMAIL=owner@rentstage.local
LOCAL_OWNER_PASSWORD=RentStage123!
LOCAL_OWNER_DISPLAY_NAME=Administrador Demo
SESSION_DURATION=12h
```

Keep your existing database configuration, particularly:

```env
POSTGRES_PORT=5433
```

## 5. Confirm version

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.6.0
```

## 6. Validate Compose

```powershell
docker compose config
```

The resulting configuration should contain four services:

```text
db
auth
api
web
```

## 7. Rebuild and start

The first v0.6 build is larger than earlier upgrades because it installs Java + Firebase CLI for the emulator and resolves the Firebase Admin/JS dependency graphs.

```powershell
docker compose build --no-cache api auth web
docker compose up -d
docker compose ps -a
```

Expected:

```text
rentstage-db-1     healthy
rentstage-auth-1   healthy
rentstage-api-1    healthy
rentstage-web-1    healthy
```

The API waits for both PostgreSQL and Authentication to become healthy.

## 8. Verify service health

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

Expected:

```text
ok
ready
```

Open the Authentication Emulator UI:

```text
http://127.0.0.1:4000
```

The Authentication tab should contain:

```text
owner@rentstage.local
```

## 9. Confirm migrations

```powershell
docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations ORDER BY version;"
```

Expected sequence:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
005_calendar_operations.sql
006_identity_access.sql
```

## 10. Confirm local identity linkage

```powershell
docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT email, identity_uid, email_verified, status FROM users ORDER BY email;"
```

Expected owner row:

```text
email                    owner@rentstage.local
identity_uid             rentstage-local-owner
email_verified           true
status                   ACTIVE
```

Confirm the membership:

```powershell
docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT t.name, u.email, tm.role, tm.status FROM tenant_memberships tm JOIN tenants t ON t.id=tm.tenant_id JOIN users u ON u.id=tm.user_id ORDER BY t.name, u.email;"
```

Expected:

```text
AudioPro Demo | owner@rentstage.local | OWNER | ACTIVE
```

## 11. Sign in

Open:

```text
http://127.0.0.1:3000/login
```

Use:

```text
Email:    owner@rentstage.local
Password: RentStage123!
```

After login, the existing v0.5 inventory, customers, quotes, reservations, calendar, and audit history should still be visible.

## 12. Run the authentication smoke test

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
```

Expected final output:

```text
RentStage v0.6 authentication smoke test passed.
```

The script verifies:

```text
Authentication emulator sign-in
CSRF bootstrap
ID-token exchange
HttpOnly server session
/auth/me
active workspace
protected dashboard access
logout
```

## 13. Functional multi-tenant test

### Create an isolated second workspace

1. Open the workspace menu in the lower-left corner.
2. Open **Administrar workspaces**.
3. Select **Crear nueva organización**.
4. Create `Studio 503` with a unique slug.
5. Enter it and confirm the business tables start empty.
6. Switch back to AudioPro Demo and confirm all original data remains.

### Test an invitation

1. In AudioPro Demo, open **Equipo**.
2. Invite a new email as `MANAGER` or `STAFF`.
3. Copy the generated local acceptance URL.
4. Open an incognito browser.
5. Create an account using exactly the invited email.
6. Open the acceptance URL and accept.
7. Confirm the workspace appears and the assigned role is active.

### Test permission boundaries

For `STAFF`, confirm:

```text
Can view operations and inventory
Can maintain customers
Can assign/check out/return equipment
Cannot create or edit quotes
Cannot create or reprogram reservations
Cannot modify catalog prices
Cannot open audit/team/organization settings
```

### Test suspension

1. As owner, suspend the invited membership.
2. In the other browser, reload a protected page.
3. Confirm the workspace is no longer authorized.

## Troubleshooting

### `auth` container is unhealthy

```powershell
docker compose logs --tail=300 auth
```

The first build must download the Firebase CLI and a Java runtime. Confirm Docker has internet access and enough disk space.

### API fails during Firebase initialization

```powershell
docker compose logs --tail=300 api auth
```

Confirm these values match:

```text
FIREBASE_PROJECT_ID=demo-rentstage
GCLOUD_PROJECT=demo-rentstage
FIREBASE_AUTH_EMULATOR_HOST=auth:9099
```

Do not include `http://` in `FIREBASE_AUTH_EMULATOR_HOST`.

### Login reports invalid credentials

Check the emulator UI at `http://127.0.0.1:4000`.

Then verify bootstrap configuration:

```powershell
docker compose exec api /app/rentstage-api healthcheck http://localhost:8080/healthz
docker compose logs --tail=200 api auth
```

To recreate only local emulator users while preserving PostgreSQL:

```powershell
docker compose down
docker volume rm rentstage_rentstage_auth_data
docker compose up -d auth api web
```

The API will bootstrap the configured local owner again. Do not remove the PostgreSQL volume.

### Browser loops between login and application

Clear local cookies for `127.0.0.1` and use the same host consistently. Avoid mixing:

```text
localhost:3000
127.0.0.1:3000
```

The default documentation uses `127.0.0.1`.

### CSRF validation failed

Reload the page. The frontend will request a new CSRF token automatically. For direct API calls, use `scripts/smoke-auth.ps1` or follow `docs/requests.http` with a cookie-aware HTTP client.

### Existing data appears missing

First confirm the active workspace in the lower-left switcher. v0.6 isolates data by membership, so a newly created workspace is expected to be empty.

### Migration failure

```powershell
docker compose logs --tail=300 api db
```

Migrations are transactional. A failed `006` does not partially commit its schema changes.

## Rollback notes

The safest rollback is code rollback while keeping the migrated database; v0.5 ignores the new identity columns/tables. However, v0.5 has no authentication protection and should remain local only.

To restore the pre-upgrade database intentionally:

```powershell
docker compose down -v
docker compose up -d db
docker compose cp .\backups\rentstage-v0.5.0-before-v0.6.dump db:/tmp/restore.dump
docker compose exec db pg_restore -U rentstage -d rentstage --clean --if-exists /tmp/restore.dump
```

Only perform a destructive restore after verifying the backup file and accepting that data created after the backup will be lost.
