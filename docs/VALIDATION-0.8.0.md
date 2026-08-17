# RentStage v0.8.0 validation record

## Scope

This record covers the **Packages Core** release built on the authenticated, role-aware, tenant-isolated RentStage v0.7.0 source tree.

Validated areas:

```text
package schema and migration ordering
package normalization and pricing rules
fixed-price cent allocation
package availability delegation and quantity limits
package API route and permission registration
role-to-permission behavior
quote-template expansion contracts
package administration routes and components
quote editor package insertion contracts
seed idempotency structure
archive composition and upgrade reproducibility
```

## Source baseline

The release source reports:

```text
VERSION                    0.8.0
web package version        0.8.0
Go source files            65
application TS/TSX files  51
migrations                 001 through 007
```

The v0.8 implementation preserves the v0.7 session, CSRF, Firebase Authentication, tenant-resolution, quote, reservation, warehouse, and audit boundaries. Package routes are added to the same authenticated modular monolith.

## Go validation

The complete API source was copied into an isolated Go 1.23 contract harness. The harness replaces only the external Firebase Admin and pgx implementations with local API-compatible stubs; the RentStage command, HTTP registration, domain services, repositories, and tests are the v0.8 source.

Commands:

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./... -count=1
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go vet ./...
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
internal/core/quote
internal/core/reservation
internal/httpapi
internal/database
```

`gofmt` was applied to all Go source before the final harness run.

## TypeScript validation

The complete v0.8 `app`, `components`, and `lib` trees were copied into the existing strict local Next/React contract harness.

Command executed:

```text
tsc -p tsconfig.validation.json --pretty false
```

Result:

```text
PASS
```

A separate syntax-only transpilation pass using the installed TypeScript compiler API also succeeded for all 51 TypeScript and TSX source files.

Validated package areas include:

- `/packages`, `/packages/new`, and `/packages/[id]`;
- package list, search, lifecycle status, pricing summaries, and responsive cards;
- package editor read/write and read-only modes;
- package availability panel;
- package permissions in `RootFrame` and `AppShell`;
- package preselection in `/quotes/new?package_id=...`;
- package-template insertion into existing quote lines;
- package audit labels and TypeScript data contracts;
- stable editor remounting when navigating between package IDs.

The browser-side merge algorithm was also reviewed against the backend quote-template contract: it preserves the incoming commercial line total when a package resource is merged into an existing quote resource line. The definitive runtime confirmation remains part of the workstation acceptance sequence.

## Formatting and configuration checks

The following checks completed successfully:

```text
gofmt over modified Go source                   PASS
git diff --check                               PASS
JSON parsing                                   PASS (3 files)
Docker Compose YAML parsing                    PASS
Compose service set: api, auth, db, web         PASS
CSS delimiter balance                          PASS
internal @/ import resolution                   PASS
private-key and local .env scan                 PASS
```

The source tree used for packaging contains no `.env`, private key, service-account key, `node_modules`, `.next`, database dump, local volume, or validation harness.

## Migration and SQL structural checks

Expected migration order:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
005_calendar_operations.sql
006_identity_access.sql
007_packages.sql
```

Result:

```text
PASS
```

Static review verifies that migration 007:

- is additive;
- creates `packages` before `package_items`;
- uses composite tenant foreign keys;
- prevents cross-tenant package/resource relationships;
- enforces tenant/slug uniqueness;
- prevents duplicate resources inside one package;
- constrains pricing mode, fixed-price consistency, positive quantities, and nonnegative prices;
- adds tenant-oriented list and lookup indexes;
- reuses the existing `set_updated_at()` trigger function.

The demo seed resolves the package and cable-kit resource through tenant-scoped natural keys and uses conflict-safe inserts. Repeated application startup therefore does not duplicate the seeded package, items, resource, or cable-kit assets.

## Authorization checks

Every package route is registered through the existing tenant-protected chain:

```text
authenticated server session
→ active RentStage user
→ active tenant membership
→ server-selected tenant context
→ package.read or package.manage
→ package handler / service / repository
```

Route permissions:

```text
GET    /api/v1/packages                               package.read
POST   /api/v1/packages                               package.manage
GET    /api/v1/packages/{packageID}                   package.read
PATCH  /api/v1/packages/{packageID}                   package.manage
DELETE /api/v1/packages/{packageID}                   package.manage
GET    /api/v1/packages/{packageID}/quote-template    package.read
POST   /api/v1/packages/{packageID}/availability      package.read
```

Role behavior:

```text
OWNER      package.read + package.manage
ADMIN      package.read + package.manage
MANAGER    package.read
STAFF      package.read
```

No package request body can select `tenant_id`. Repositories receive the active tenant only from authenticated server context.

## Commercial and availability invariants

Packages reference resources and quantities rather than serialized physical assets:

```text
package item                 physical fulfillment
2 × Shure SM58               exact microphone units assigned later
```

Quote expansion produces ordinary quote-item snapshots. Later changes to package composition or pricing cannot mutate a previously saved quote.

The package repository reads summary totals and ordered item lines in one read-only `REPEATABLE READ` transaction. Static compilation confirms the pgx transaction contract, and the final PostgreSQL runtime check remains part of the workstation acceptance sequence.

For fixed-price packages:

```text
sum(expanded line totals) + package extra charges
= exact effective package price × requested package quantity
```

Discount allocation is cent-based and never produces a negative line total. A fixed price above component value is represented as quote-level extra charges.

Package availability multiplies each component quantity by the requested number of packages and delegates to the existing reservation-aware availability service. The package service computes the maximum package quantity that keeps every delegated resource request within the availability engine's 10,000-unit bound and returns a field validation error when the request exceeds that limit.

Quotes remain nonblocking. Quote-to-reservation conversion remains the authoritative final availability check.

## Packaging checks

The final release process verifies:

```text
upgrade ZIP contains only v0.7 → v0.8 changes       PASS
full ZIP contains one rentstage-starter/ root        PASS
upgrade ZIP excludes .env and generated content      PASS
full ZIP excludes .git and generated content         PASS
upgrade ZIP CRC                                      PASS
full ZIP CRC                                         PASS
upgrade overlay equals full v0.8 source tree          PASS
SHA-256 generation and verification                  PASS
```

The byte-for-byte overlay comparison starts from the distributed v0.7.0 full archive, applies the v0.8.0 upgrade archive, and compares the resulting release tree with the v0.8.0 full archive.

## Runtime integration still required

The packaging environment does not provide Docker Engine, PostgreSQL 18, or the production Go 1.26.5 / Next.js 16.3 dependency build. The authoritative integration test must therefore run on the RentStage workstation:

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a

Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz

powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
```

Then confirm `007_packages.sql` in `schema_migrations` and complete the manual acceptance sequence in `UPGRADE-0.8.0.md`.

## Boundaries intentionally left open

- No anonymous package or catalog endpoint is introduced in v0.8.
- Package availability is advisory until reservation conversion.
- Saved quotes preserve snapshots but do not yet store package-lineage reporting fields.
- Space, staffing, and service-provider capacity need later availability strategies.
- Package images remain external URLs rather than managed uploads.
- Tenant storefronts, public quote requests, online customer acceptance, payments, WhatsApp, and AI orchestration remain deferred.
