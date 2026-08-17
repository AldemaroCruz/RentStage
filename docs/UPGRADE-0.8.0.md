# Upgrade RentStage v0.7.0 → v0.8.0

## Scope

RentStage v0.8.0 introduces **Packages Core**. This is a full-stack upgrade: it adds PostgreSQL migration `007_packages.sql`, a new Go domain module and API routes, package permissions, package-management pages, availability validation, quote-template expansion, demo data, audit labels, and smoke tests.

The upgrade preserves:

- existing tenants, users, memberships, invitations, roles, and workspace preferences;
- categories, resources, physical assets, customers, quotes, and reservations;
- warehouse assignments, status history, schedule history, and audit events;
- the existing `.env` file;
- PostgreSQL and Firebase Authentication emulator volumes.

The public catalog, anonymous quote request, tenant public page, and online customer acceptance are not included in v0.8.0.

## New migration

```text
007_packages.sql
```

It creates:

```text
packages
package_items
```

The migration is additive. It does not delete or rewrite existing commercial, operational, or identity records.

## 1. Back up PostgreSQL

Run from PowerShell in the existing project directory:

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

New-Item -ItemType Directory -Force .\backups | Out-Null

docker compose exec db `
  pg_dump `
  -U rentstage `
  -d rentstage `
  -Fc `
  -f /tmp/rentstage-v0.7.0-before-v0.8.0.dump

docker compose cp `
  db:/tmp/rentstage-v0.7.0-before-v0.8.0.dump `
  .\backups\rentstage-v0.7.0-before-v0.8.0.dump

Get-Item .\backups\rentstage-v0.7.0-before-v0.8.0.dump
```

Do not continue unless the dump exists and has a nonzero size.

## 2. Stop services without deleting volumes

```powershell
docker compose down
```

Do **not** run:

```powershell
docker compose down -v
```

The `-v` option deletes PostgreSQL and Authentication emulator data.

## 3. Apply the upgrade archive

```powershell
Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.8.0.zip" `
  -DestinationPath . `
  -Force
```

The upgrade archive intentionally excludes `.env`, generated build directories, database dumps, and local volumes.

Confirm the source version:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.8.0
```

## 4. Build API and web

Both application images changed:

```powershell
docker compose build --no-cache api web
```

Start the complete stack:

```powershell
docker compose up -d
docker compose ps -a
```

Wait until `db`, `auth`, and `api` are healthy and `web` is running.

## 5. Verify health and migration

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz

(Invoke-WebRequest `
  http://127.0.0.1:3000/login `
  -UseBasicParsing).StatusCode
```

Expected values:

```text
healthz  status = ok
readyz   status = ready
/login   HTTP 200
```

Confirm migration 007:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations WHERE version = '007_packages.sql';"
```

Expected: one row for `007_packages.sql`.

Optional structural verification:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "\dt packages" `
  -c "\dt package_items"
```

## 6. Run automated smoke tests

Authentication and tenant boundary:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
```

Packages Core:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
```

The package smoke test validates the seeded package, package detail, two-package quote expansion, exact commercial total, and a reservation-aware availability request.

## 7. Manual acceptance

Sign in with the configured local owner account and open:

```text
http://127.0.0.1:3000/packages
```

With the default local seed, confirm **Paquete Fiesta 100 personas** contains:

```text
2 × JBL PRX815W
2 × QSC KS118
1 × Behringer X32 Compact
2 × Shure SM58
1 × Kit de cableado básico
```

Then verify:

1. Create a `SUM_ITEMS` package and confirm its selling price follows the component total.
2. Change it to `FIXED` and confirm RentStage displays either a commercial saving or additional margin.
3. Test a free period in **Probar disponibilidad**.
4. Archive and reactivate a package.
5. Open `/quotes/new`, select a customer, add the package, and confirm the resources expand into ordinary quote lines.
6. Save the quote, change the package later, and confirm the saved quote retains its historical lines and prices.
7. Switch workspaces and confirm package isolation.
8. Confirm OWNER/ADMIN can manage packages while MANAGER/STAFF have read-only package access.
9. Open **Auditoría** and confirm package mutation events use the authenticated actor.

## Demo seed behavior

When `SEED_DEMO_DATA=true`, the API seed ensures the cable-kit resource, three available cable-kit assets, and the Fiesta 100 package are present. The seed uses tenant-scoped natural lookups and conflict-safe inserts so repeated starts remain idempotent.

When `SEED_DEMO_DATA=false`, migration 007 still applies, but the demo package is not created. Create the first package through the UI before running `smoke-packages.ps1`, or pass that script an existing package slug.

## Rollback

### Application-only rollback

To return the application source to v0.7.0 while preserving current data:

1. stop the stack;
2. restore the v0.7.0 application files;
3. rebuild `api` and `web`;
4. start the stack again.

The unused additive `packages` and `package_items` tables may remain in PostgreSQL without affecting v0.7.0.

### Complete rollback

To remove all v0.8 schema and package data, restore the pre-upgrade database dump into a clean PostgreSQL volume. This replaces the current database with its v0.7.0 state, so preserve any post-upgrade changes separately before restoring.
