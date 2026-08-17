# Upgrade RentStage v0.9.0 → v0.10.0

## Scope

RentStage v0.10.0 introduces **Quote Portal**. This is an additive full-stack update containing only files added or changed since v0.9.0.

```text
portal settings per tenant
expiring/revocable quote links
public formal quote document
online customer acceptance or rejection
versioned terms and response evidence
final availability validation
atomic acceptance → PENDING reservation
```

The update preserves `.env`, PostgreSQL data, Firebase Authentication emulator data, tenants, memberships, public catalogs, packages, quote requests, customers, quotes, reservations, warehouse history, and audit history.

## New migration

```text
009_quote_portal.sql
```

It creates:

```text
quote_portal_settings
quote_portals
quote_portal_events
```

The migration stores only SHA-256 bearer-token hashes and adds an administrator quote-status synchronization trigger. It does not delete or rewrite existing business records.

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
  -f /tmp/rentstage-v0.9.0-before-v0.10.0.dump

docker compose cp `
  db:/tmp/rentstage-v0.9.0-before-v0.10.0.dump `
  .\backups\rentstage-v0.9.0-before-v0.10.0.dump

Get-Item .\backups\rentstage-v0.9.0-before-v0.10.0.dump
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
  -Path "$HOME\Downloads\rentstage-upgrade-v0.10.0.zip" `
  -DestinationPath . `
  -Force
```

The archive intentionally excludes `.env`, generated build directories, database dumps, local volumes, and credentials.

Confirm the source version:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.10.0
```

## 4. Keep the existing fingerprint salt

Version 0.10 reuses the `PUBLIC_REQUEST_FINGERPRINT_SALT` introduced by v0.9 for privacy-preserving Quote Portal origin evidence.

Your `.env` should already contain a local value such as:

```env
PUBLIC_REQUEST_FINGERPRINT_SALT=rentstage-local-public-catalog
```

Staging and production must continue using an explicit random value of at least 32 characters stored in Secret Manager or the deployment environment. No additional v0.10 environment variable is required.

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

Wait until `db`, `auth`, and `api` are healthy and `web` is running. The API applies migration 009 automatically during startup.

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

Confirm migration 009:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations WHERE version = '009_quote_portal.sql';"
```

Expected: one row for `009_quote_portal.sql`.

Verify the new tables:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "\dt quote_portal_settings" `
  -c "\dt quote_portals" `
  -c "\dt quote_portal_events"
```

Optional token-storage verification after sending a quote:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT status, revision, length(token_hash) AS hash_length, expires_at FROM quote_portals ORDER BY created_at DESC LIMIT 5;"
```

`hash_length` must be `64`. There is intentionally no raw-token column.

## 7. Run regression and Quote Portal smoke tests

Authentication and tenant boundary:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
```

Packages Core regression:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
```

Public Catalog regression:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1 -SkipSubmission
```

Quote Portal end-to-end flow:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-quote-portal.ps1
```

The Quote Portal script:

- signs in as the local owner;
- checks portal settings and quote/reservation permissions;
- finds a free future period;
- creates and sends a temporary quote;
- confirms the bearer secret exists only in `/q#<token>`;
- reads the anonymous document through `X-RentStage-Quote-Token`;
- rotates the token and verifies the old token returns HTTP 404;
- accepts the quote through the CSRF-protected public endpoint;
- verifies acceptance idempotency;
- verifies quote, reservation, portal evidence, and decision source;
- cancels the temporary reservation to release capacity;
- closes the administrator session.

## 8. Manual acceptance

### Configure the portal

Sign in and open:

```text
http://127.0.0.1:3000/settings/quote-portal
```

Verify:

1. The portal can be enabled and disabled.
2. Title, introduction, accent color, validity, rejection policy, response-name policy, terms text, and terms version save correctly.
3. A role with `quote.read` but without `quote.manage` can inspect settings but cannot save changes.
4. Disabling the portal prevents new issuance and makes existing public links unavailable; re-enabling restores still-active links.

### Send and view a quote

1. Create a `DRAFT` quote for a future free period.
2. Open its detail page and select **Enviar y generar enlace**.
3. Copy the one-time link before refreshing.
4. Confirm its shape is `/q#<secret>` with no query parameter or token path segment.
5. Open it in an incognito window.
6. Confirm the browser removes the fragment from the visible address after loading.
7. Confirm tenant identity, customer, event period, lines, prices, total, terms, and expiry render without login.
8. Refresh the same tab and confirm tab-scoped access remains available.

### Accept online

1. Enter the response name and optional email.
2. Accept the displayed terms version.
3. Confirm the page reports `ACCEPTED` and shows a reservation number.
4. Return to the admin quote and verify:
   - quote status is `ACCEPTED`;
   - a linked `PENDING` reservation exists;
   - portal status/source is `ACCEPTED` / `CUSTOMER`;
   - response name, email, terms version, view data, and decision timestamp are present;
   - the raw link is not returned by an ordinary quote reload.
5. Repeat the public accept request and confirm no second reservation is created.

### Reject, rotate, revoke, and expire

1. Send another quote and reject it with an optional reason.
2. Confirm the quote and portal become `REJECTED` without creating a reservation.
3. Send/reissue another quote, rotate its link, and verify the previous secret returns HTTP 404.
4. Revoke the active link and verify the public page closes while the quote remains `SENT`.
5. Generate a new link from the same sent quote.
6. Use a short/past expiry in a test environment and verify the quote/portal become `EXPIRED` when accessed.

### Availability conflict

1. Send a quote for currently free inventory.
2. Create a conflicting reservation that consumes the same capacity.
3. Attempt customer acceptance.
4. Confirm HTTP 409 and a privacy-reduced list containing only resource names, requested quantities, and `can_fulfill`.
5. Confirm the quote remains `SENT`, the portal remains `ACTIVE`, and no reservation is created from the blocked acceptance.

## Rollback

### Application-only rollback

To return application source to v0.9.0 while preserving current data:

1. stop the stack;
2. restore the v0.9.0 application files;
3. rebuild `api` and `web`;
4. start the stack again.

The unused additive Quote Portal tables and trigger may remain, but an older application may still change quote statuses and therefore execute the synchronization trigger. For the cleanest application-only rollback, disable or drop `sync_quote_portal_status` after reviewing operational impact.

### Complete rollback

To remove the v0.10 schema and all Quote Portal evidence, restore the pre-upgrade dump into a clean PostgreSQL volume. This returns the complete database to its v0.9 state and discards changes made after the backup.
