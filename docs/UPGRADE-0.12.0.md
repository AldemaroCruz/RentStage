# Upgrade RentStage v0.11.0 → v0.12.0

## Scope

RentStage v0.12.0 introduces the **El Salvador DTE integration foundation** as an additive update containing only files added or changed since v0.11.0.

```text
invoice fiscal snapshot
DTE preparation and control numbering
local MOCK signer/reception provider
configurable MH_HTTP adapter
manual submission and retry
receipt-seal evidence
controlled invalidation
fiscal inbox and event history
secret references instead of stored credentials
```

The update preserves `.env`, PostgreSQL data, Firebase Authentication emulator data, tenants, memberships, catalogs, packages, public quote requests, Quote Portal evidence, customers, quotes, reservations, invoices, payments, deposits, warehouse history, and audit history.

> The default `MOCK / TEST` provider is a local lifecycle simulator. Its seal has no fiscal validity. `MH_HTTP` is a configurable adapter that must be aligned and certified with the current DGII schemas, endpoints, credentials, signing mechanism, and onboarding cases assigned to the taxpayer before any real transmission.

## New migration

```text
011_dte_integration.sql
```

It adds DTE receiver/location fields, immutable invoice fiscal snapshots, DTE line metadata, and creates:

```text
dte_settings
dte_documents
dte_events
```

The migration is additive. It creates default `MOCK / TEST` settings for existing tenants and does not transmit, sign, invalidate, or alter any existing invoice automatically.

## 1. Back up PostgreSQL

Run from PowerShell in the project directory:

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

New-Item -ItemType Directory -Force .\backups | Out-Null

docker compose exec db `
  pg_dump `
  -U rentstage `
  -d rentstage `
  -Fc `
  -f /tmp/rentstage-v0.11.0-before-v0.12.0.dump

docker compose cp `
  db:/tmp/rentstage-v0.11.0-before-v0.12.0.dump `
  .\backups\rentstage-v0.11.0-before-v0.12.0.dump

Get-Item .\backups\rentstage-v0.11.0-before-v0.12.0.dump
```

## 2. Stop services without deleting volumes

```powershell
docker compose down
```

Do **not** run:

```powershell
docker compose down -v
```

`-v` removes the PostgreSQL and Authentication-emulator volumes.

## 3. Apply the incremental archive

Extract directly into the existing project root:

```powershell
Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.12.0.zip" `
  -DestinationPath . `
  -Force
```

Confirm the release:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.12.0
```

The archive does not contain `.env`.

## 4. Environment configuration

No credential is required for the default local provider:

```text
provider_mode  MOCK
environment    TEST
```

For `MH_HTTP`, RentStage stores only references such as:

```text
env://DTE_MH_USER
env://DTE_MH_PASSWORD
env://DTE_MH_SIGNING_PASSWORD
```

Add the corresponding values to the local `.env` only when using an official test integration:

```env
DTE_MH_USER=
DTE_MH_PASSWORD=
DTE_MH_SIGNING_PASSWORD=
```

For Cloud Run, inject those variables from Secret Manager. Never paste a real password or private signing value into the RentStage settings page or PostgreSQL.

## 5. Build API and web

DTE changes affect PostgreSQL, Go, and Next.js:

```powershell
docker compose build --no-cache api web

docker compose up -d

docker compose ps -a
```

The API migration runner applies `011_dte_integration.sql` during startup.

## 6. Verify health and migration

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz

(Invoke-WebRequest `
  http://127.0.0.1:3000/login `
  -UseBasicParsing).StatusCode
```

Expected results are `ok`, `ready`, and HTTP `200`.

Confirm the migration:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations WHERE version = '011_dte_integration.sql';"
```

Confirm the new tables:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "\dt dte_settings" `
  -c "\dt dte_documents" `
  -c "\dt dte_events"
```

Confirm that only secret references, not secret values, are persisted:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT provider_mode, environment, user_secret_ref, password_secret_ref, signing_password_secret_ref FROM dte_settings;"
```

In local MOCK mode the references should be empty.

## 7. Run regression and DTE smoke tests

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1 -SkipSubmission
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-quote-portal.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-billing.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-dte.ps1
```

The DTE smoke test validates:

```text
fiscal permissions
issuer and receiver profile snapshots
MOCK / TEST provider configuration
invoice READY_FOR_DTE state
DTE control and generation codes
immutable JSON payload
absence of credential material in the payload
manual sign/transmit lifecycle
MOCK receipt seal
invoice fiscal ACCEPTED state
DTE invalidation
invoice fiscal VOIDED state
fiscal inbox visibility
audited cleanup and configuration restoration
```

The test consumes one invoice sequence and one DTE control sequence. Both records are retained as voided/invalidated history and their numbers are intentionally not reused.

## 8. Manual acceptance

### Configure DTE

Open:

```text
http://127.0.0.1:3000/settings/dte
```

Keep `MOCK / TEST` for local validation. Review establishment code, point of sale, schema version, retry policy, and default document type.

### Complete issuer and receiver data

Open:

```text
http://127.0.0.1:3000/settings/billing
http://127.0.0.1:3000/customers
```

Complete NIT, NRC, economic activity, fiscal address, geographic codes, email, and phone. For a CCF (`03`), the receiver also requires its registration and economic-activity data.

### Prepare and submit

1. Create and issue an internal invoice.
2. Open its detail page.
3. Select **Preparar DTE**.
4. Review the immutable JSON snapshot.
5. Select **Firmar y transmitir**.
6. Confirm `ACCEPTED`, one attempt, and a `MOCK-...` receipt seal.
7. Open `/dte` and verify the fiscal inbox and event history.
8. Invalidate the test DTE and confirm the original payload and provider evidence remain available.

## 9. Official-provider boundary

Do not select `MH_HTTP / PRODUCTION` until the taxpayer has completed the official incorporation and certification process. Before enabling production, verify at minimum:

```text
taxpayer authorization as an electronic issuer
current JSON schemas and catalog versions
official test and production service URLs
signer requirements and private-key handling
authentication credentials
required certification cases
establishment and point-of-sale codes
invalidation and contingency rules
legible representation requirements
public-query/QR requirements
```

RentStage rejects production HTTP URLs, URL-embedded credentials, localhost, loopback, private, link-local, metadata, and other non-public endpoint targets. This reduces SSRF exposure from tenant-configurable fiscal URLs.

## Rollback

### Application-only rollback

If migration 011 succeeded but the application must temporarily return to v0.11, restore the previous source tree and images without deleting the database. v0.11 ignores the additive DTE tables and fields.

### Complete rollback

To remove all v0.12 schema/data, restore the pre-upgrade dump into a clean database volume. Do not manually delete DTE tables from a production-like database because control numbers, payloads, seals, invalidations, and audit events are historical evidence.
