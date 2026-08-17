# RentStage v0.9.0 validation record

## Scope

This record covers the **Public Catalog** release built on the authenticated, role-aware, tenant-isolated RentStage v0.8.0 source tree.

Validated areas:

```text
public catalog schema and migration ordering
public package/resource publication rules
anonymous catalog visibility
anonymous CSRF-protected writes
privacy-reduced availability responses
quote-request normalization and anti-abuse fingerprinting
historical package, item, price, currency, availability, and terms snapshots
request status workflow and request-to-quote transaction
role-to-permission behavior
public and administrative Next.js routes
update-archive integrity and v0.8 overlay reproducibility
```

## Source baseline

The release source reports:

```text
VERSION                    0.9.0
web package version        0.9.0
migrations                 001 through 008
update baseline            distributed v0.8.0 source
```

Version 0.9 preserves the v0.8 Firebase Authentication, server session, CSRF, tenant-resolution, package, quote, reservation, warehouse, and audit boundaries.

## Go validation

The complete API source was copied into an isolated Go 1.23 contract harness. The harness replaces only the external Firebase Admin and pgx implementations with local API-compatible stubs; RentStage command wiring, route registration, domain services, repositories, and tests are the v0.9 source.

Commands:

```bash
GOTOOLCHAIN=local GOWORK=off go test ./...
GOTOOLCHAIN=local GOWORK=off go vet ./...
```

Result:

```text
PASS
```

Passing packages include:

```text
internal/authn
internal/core/availability
internal/core/customer
internal/core/identity
internal/core/operations
internal/core/packages
internal/core/publiccatalog
internal/core/quote
internal/core/reservation
internal/httpapi
internal/database
```

`gofmt` was applied to all Go source before the final harness run.

New public-catalog tests cover:

- settings normalization and unsafe URL/color rejection;
- quote-request period, contact, consent, and package-selection validation;
- strict email parsing;
- tenant-scoped salted request fingerprints;
- privacy reduction of availability results;
- Unicode-safe truncation and public slug normalization.

## TypeScript validation

The complete v0.9 `app`, `components`, and `lib` trees were copied into a strict local React/Next contract harness.

Command:

```text
tsc -p tsconfig.json --pretty false
```

Result:

```text
PASS
```

A separate TypeScript compiler-API transpilation pass also validates every application `.ts` and `.tsx` file for syntax.

Validated routes/components include:

```text
/settings/public-catalog
/quote-requests
/quote-requests/[id]
/p/[tenantSlug]
/p/[tenantSlug]/packages/[packageSlug]
/p/[tenantSlug]/resources/[resourceSlug]
/p/[tenantSlug]/request
PublicCatalogShell
RootFrame public-route boundary
AppShell permission-filtered navigation
same-origin API proxy header propagation
```

## Security and isolation review

Static review confirms:

- disabled catalogs return the same public not-found boundary as missing catalogs;
- package/resource public reads require active, explicitly published same-tenant records;
- anonymous writes still pass the global CSRF middleware;
- public availability omits resource IDs, asset IDs, serials, tags, eligible-asset counts, reserved quantities, and available quantities;
- raw client IP addresses are not persisted;
- the request fingerprint combines a deployment salt, tenant ID, and normalized client origin through SHA-256;
- five persisted requests per tenant/fingerprint/hour are allowed before HTTP 429; the definitive count is serialized with a transaction-scoped PostgreSQL advisory lock to close concurrent-submission races;
- the honeypot returns a generic receipt without persistence;
- anonymous request audit records use an `API` actor rather than a forged user actor;
- request administration always derives `tenant_id` from authenticated server context;
- package/resource publication updates cannot cross tenant boundaries;
- request conversion locks the request and writes customer, quote, quote items, and conversion state in one transaction;
- converted or closed/spam requests cannot be converted through the supported endpoint;
- terms text and terms version are snapshotted at submission time.

## Migration and SQL structural checks

Expected order:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
005_calendar_operations.sql
006_identity_access.sql
007_packages.sql
008_public_catalog.sql
```

Static migration review confirms migration 008:

- creates tenant-owned public-catalog settings;
- adds publication fields and consistency checks to packages/resources;
- creates tenant-unique public resource slugs;
- creates quote-request header, package-snapshot, and line-snapshot tables;
- uses composite tenant foreign keys for request relationships;
- enforces contact, consent, period, money, currency, and conversion consistency;
- adds status/rate-limit/read indexes and update triggers;
- preserves all v0.8 records.

The definitive DDL execution remains part of the PostgreSQL 18 workstation test.

## Formatting and configuration checks

The final release process checks:

```text
gofmt over Go source                         PASS
git diff --check                            PASS
JSON parsing                                PASS
Docker Compose YAML parsing                 PASS
CSS delimiter balance                       PASS
internal @/ import resolution                PASS
migration order                              PASS
private-key and local .env scan              PASS
PowerShell smoke-script structural review    PASS
```

The update archive excludes `.env`, `.git`, `node_modules`, `.next`, database dumps, backups, local volumes, and validation harnesses.

## Update archive checks

The release distributes only:

```text
rentstage-upgrade-v0.9.0.zip
rentstage-v0.9.0.sha256
```

Packaging validation performs:

```text
ZIP CRC integrity                           PASS
update paths rooted at project-relative     PASS
private/generated file exclusion            PASS
v0.8 archive + v0.9 update overlay          PASS
byte-for-byte match with v0.9 source         PASS
SHA-256 generation and verification          PASS
```

## Runtime integration still required

The packaging environment does not provide Docker Engine or PostgreSQL 18. The authoritative integration test must run on the RentStage workstation:

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a

Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz

powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1
```

Then confirm `008_public_catalog.sql` in `schema_migrations` and complete the manual acceptance sequence in `UPGRADE-0.9.0.md`.

## Boundaries intentionally left open

- Quote requests do not block inventory and are not reservations.
- Conversion creates a draft quote; formal quote acceptance remains an internal admin action in v0.9.
- Online acceptance/rejection tokens, versioned customer signatures, and acceptance-to-reservation orchestration are deferred to v0.10.
- Public images remain external URLs rather than managed uploads.
- CAPTCHA, managed WAF/rate limiting, transactional email, payments, WhatsApp, and AI orchestration remain deployment/product follow-ons.
