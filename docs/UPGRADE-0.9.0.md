# Upgrade RentStage v0.8.0 → v0.9.0

## Scope

RentStage v0.9.0 introduces **Public Catalog**. This is an additive full-stack upgrade with:

```text
tenant storefronts
public package/resource publication
anonymous availability checks
anonymous quote requests
administrative request inbox
request → customer + DRAFT quote conversion
```

The upgrade preserves the existing `.env`, PostgreSQL data, Firebase Authentication emulator data, tenants, memberships, packages, resources, customers, quotes, reservations, warehouse history, and audit history.

Online acceptance of a formal quote is not part of v0.9.0. It remains planned for v0.10.0.

## New migration

```text
008_public_catalog.sql
```

It creates:

```text
public_catalog_settings
quote_requests
quote_request_packages
quote_request_items
```

It also adds public-presentation columns to `packages` and `resources`. The migration does not delete or rewrite existing business records.

## 1. Back up PostgreSQL

Run from PowerShell in the current project directory:

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

New-Item -ItemType Directory -Force .\backups | Out-Null

docker compose exec db `
  pg_dump `
  -U rentstage `
  -d rentstage `
  -Fc `
  -f /tmp/rentstage-v0.8.0-before-v0.9.0.dump

docker compose cp `
  db:/tmp/rentstage-v0.8.0-before-v0.9.0.dump `
  .\backups\rentstage-v0.8.0-before-v0.9.0.dump

Get-Item .\backups\rentstage-v0.8.0-before-v0.9.0.dump
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

## 3. Apply the update archive

```powershell
Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.9.0.zip" `
  -DestinationPath . `
  -Force
```

The archive contains only files added or changed since v0.8.0. It intentionally excludes `.env`, generated build directories, database dumps, and local volumes.

Confirm the source version:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.9.0
```

## 4. Add the anti-abuse salt to `.env`

Your existing `.env` is preserved by the update. Add this setting when it is not already present:

```env
PUBLIC_REQUEST_FINGERPRINT_SALT=rentstage-local-public-catalog
```

That value is acceptable only for local development. Staging and production must use an explicit random value of at least 32 characters, for example:

```powershell
$bytes = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
[Convert]::ToHexString($bytes)
```

Store the generated value in Secret Manager or the deployment environment, not in source control.

## 5. Build API and web

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

The API applies migration 008 automatically during startup.

## 6. Verify health and migration

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz

(Invoke-WebRequest `
  http://127.0.0.1:3000/login `
  -UseBasicParsing).StatusCode
```

Expected:

```text
healthz  status = ok
readyz   status = ready
/login   HTTP 200
```

Confirm migration 008:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations WHERE version = '008_public_catalog.sql';"
```

Expected: one row for `008_public_catalog.sql`.

Optional structural verification:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "\dt public_catalog_settings" `
  -c "\dt quote_requests" `
  -c "\dt quote_request_packages" `
  -c "\dt quote_request_items"
```

## 7. Run smoke tests

Authentication and tenant boundary:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
```

Packages Core regression:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
```

Public Catalog and quote-request flow:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1
```

The public-catalog script:

- loads the anonymous tenant storefront;
- reads a published package;
- performs an anonymous CSRF-protected availability check;
- verifies that internal inventory identifiers are absent;
- submits a quote request;
- signs in as the local owner;
- finds the request in the tenant inbox;
- verifies package, quote-line, availability, currency, terms, and consent snapshots;
- marks the smoke request as `SPAM` to keep the active inbox clean.

Use the read-only mode when you do not want to create a request:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1 -SkipSubmission
```

## 8. Manual acceptance

Sign in and open:

```text
http://127.0.0.1:3000/settings/public-catalog
```

Verify:

1. The public catalog is enabled.
2. **Paquete Fiesta 100 personas** is visible and featured.
3. The seeded resources have public slugs and descriptions.
4. Headline, contact information, accent color, terms, and price visibility can be saved.
5. An archived or incomplete package cannot be published.
6. A resource cannot be featured unless it is visible.

Open the anonymous storefront in a private/incognito window:

```text
http://127.0.0.1:3000/p/audiopro-demo
```

Verify:

1. The page opens without authentication.
2. Only explicitly published packages/resources appear.
3. Hiding prices removes public amounts without removing internal prices.
4. Package and resource detail pages work.
5. The quote-request form checks the selected period and submits a reference code.
6. The public availability response does not expose asset IDs, tags, serial numbers, eligible-asset counts, or available quantities.

Return to the authenticated panel:

```text
http://127.0.0.1:3000/quote-requests
```

Verify:

1. The request appears in the correct tenant only.
2. Detail preserves the public terms text/version accepted at submission time.
3. Status changes among `NEW`, `IN_REVIEW`, `CLOSED`, and `SPAM` are audited.
4. Conversion creates or reuses a same-tenant customer and creates a `DRAFT` quote.
5. The converted quote retains the request's historical lines and exact commercial total.
6. The public request itself never blocks inventory or creates a reservation.

## Demo seed behavior

With `SEED_DEMO_DATA=true`, the seed initializes the AudioPro Demo storefront only when `public_catalog_settings` does not yet exist for that tenant. This gives a useful first-run catalog while preserving every later administrator edit.

The seed publishes the Fiesta 100 package and selected demo resources. With `SEED_DEMO_DATA=false`, migration 008 still applies, but catalogs remain disabled until configured through the admin UI.

## Rollback

### Application-only rollback

To return application source to v0.8.0 while preserving current data:

1. stop the stack;
2. restore the v0.8.0 application files;
3. rebuild `api` and `web`;
4. start the stack again.

The unused additive public-catalog tables and columns may remain without affecting the v0.8 application.

### Complete rollback

To remove all v0.9 schema and request data, restore the pre-upgrade dump into a clean PostgreSQL volume. This returns the complete database to its v0.8 state and discards changes made after the backup.
