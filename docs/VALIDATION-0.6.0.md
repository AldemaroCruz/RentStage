# RentStage v0.6.0 validation record

## Scope

This record covers the source package prepared for the v0.5 → v0.6 identity/access upgrade.

The validation focused on:

```text
Go compilation contracts
Go unit tests and vet
TypeScript strict contracts
route and permission consistency
migration ordering
configuration parsing
archive integrity
upgrade reproducibility
security invariants that can be checked without a live runtime
```

## Go validation

The project source was copied into an isolated Go 1.23 static harness. The harness replaces only external Firebase Admin and pgx implementations with local API-compatible stubs; all RentStage packages and tests are the real v0.6 source.

Commands:

```bash
GOTOOLCHAIN=local GOWORK=off go test ./...
GOTOOLCHAIN=local GOWORK=off go vet ./...
```

Result:

```text
PASS
```

Packages with passing tests include:

```text
internal/authn
internal/core/identity
internal/core/availability
internal/core/customer
internal/core/operations
internal/core/quote
internal/core/reservation
```

### New v0.6 test coverage

- CSRF token/cookie equality and rejection cases.
- Session cookie attributes and clearing behavior.
- Role-to-permission mapping.
- OWNER/ADMIN membership hierarchy.
- Organization normalization and validation.
- Invitation email and role validation.
- Existing customer, quote, availability, booking, warehouse, calendar, and operations tests.

## TypeScript validation

The complete v0.6 `app`, `components`, and `lib` trees were copied into the existing strict Next/React contract harness.

Command:

```bash
npx tsc -p tsconfig.validation.json --pretty false
```

Result:

```text
PASS
```

Validated areas include:

- `AuthProvider` session lifecycle.
- Firebase client initialization and emulator connection.
- login, signup, onboarding, workspace selector, invitation, profile, team, and organization pages.
- permission-aware navigation.
- route-level permission guards.
- permission-aware quote, reservation, customer, catalog, inventory, calendar, and warehouse controls.
- CSRF-aware API client.
- cookie-preserving Next.js proxy.

## Formatting and syntax

- `gofmt` applied to all Go source files.
- JSON files parsed successfully.
- Docker Compose YAML parsed successfully with a YAML parser.
- shell entrypoint passed `sh -n`.
- PowerShell smoke script was reviewed for strict error handling and cookie-preserving request flow.
- CSS braces and TS/TSX imports were checked programmatically.

## Migration checks

Expected order:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
005_calendar_operations.sql
006_identity_access.sql
```

Static migration checks verified:

- `006` is append-only relative to the existing commercial/operations tables.
- organization identity columns use `ADD COLUMN IF NOT EXISTS`.
- invitation and preference tables use `CREATE TABLE IF NOT EXISTS`.
- membership statuses are constrained.
- identity UID and normalized email indexes are unique.
- only one pending invitation per tenant/email is allowed.
- update timestamp triggers reference the existing `set_updated_at()` function.
- the demo seed upserts the owner identity/membership without deleting existing data.

## Authorization checks

The route table was checked so every business endpoint is registered through:

```text
authentication middleware
→ tenant membership middleware
→ permission middleware
→ handler
```

Public endpoints are intentionally limited to:

```text
GET  /healthz
GET  /readyz
GET  /api/v1/auth/csrf
POST /api/v1/auth/session
DELETE /api/v1/auth/session
GET  /api/v1/invitations/{token}
```

`POST /auth/session` and `DELETE /auth/session` remain subject to CSRF validation.

Protected non-tenant endpoints are limited to:

```text
GET  /api/v1/auth/me
POST /api/v1/auth/select-tenant
POST /api/v1/organizations
POST /api/v1/invitations/{token}/accept
```

They require an authenticated session but do not require an existing tenant, which is necessary for onboarding and invitation acceptance.

## Tenant-isolation checks

Static review confirmed:

- no protected business route trusts a browser `tenant_id` body field as context;
- the tenant cookie is resolved only against `ListWorkspaces`, which returns active memberships to active tenants;
- role and tenant are placed into server request context;
- existing repositories continue to query by context `tenant_id`;
- composite tenant foreign keys from earlier migrations remain unchanged;
- the web proxy no longer injects the former local tenant/actor headers.

## Cookie and CSRF checks

Session cookie:

```text
HttpOnly=true
SameSite=Lax
Secure=configurable
Path=/
MaxAge=SESSION_DURATION
```

Tenant cookie:

```text
HttpOnly=true
SameSite=Lax
membership-validated server-side
```

CSRF cookie:

```text
HttpOnly=true
SameSite=Strict
random 256-bit token
constant-time header comparison
```

Every unsafe `/api/` method is rejected when the CSRF cookie/header pair is absent or different.

## Local smoke-test artifact

Included:

```text
scripts/smoke-auth.ps1
```

It is designed to validate against a running Docker stack:

```text
API readiness
Firebase emulator password sign-in
CSRF token and cookie
ID-token → HttpOnly session exchange
/auth/me
protected dashboard access
logout
```

## Package checks

Before release packaging, the following checks are performed:

- full ZIP contains the complete v0.6 source;
- upgrade ZIP contains only changes from v0.5;
- upgrade ZIP excludes `.env`;
- archives exclude `node_modules`, `.next`, databases, backups, and harnesses;
- applying the upgrade ZIP over a clean v0.5 tree produces the same source tree as the v0.6 full package;
- archives pass CRC integrity checks;
- SHA-256 hashes are generated and verified.

## Runtime validation limitation

This execution environment does not provide Docker Engine, a running PostgreSQL instance, or outbound package resolution for a real build. Therefore it was not possible here to execute:

```text
docker compose build api auth web
docker compose up
migration 006 against PostgreSQL 18
Firebase Authentication emulator startup
real Firebase Admin session-cookie exchange
real Next.js production build with downloaded packages
```

Those integration checks must run on the user's Docker Desktop environment. The included upgrade guide and PowerShell smoke test provide the exact validation sequence.

This limitation is distinct from static correctness: the RentStage source, internal tests, type contracts, configuration structure, migration ordering, and package reproducibility were validated locally.
