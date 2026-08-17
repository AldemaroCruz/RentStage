# RentStage v0.12.0 validation record

## Scope

This record covers the incremental v0.11.0 → v0.12.0 El Salvador DTE integration update.

Validated areas:

```text
Go compilation contracts, tests, and vet
TypeScript strict contracts and TS/TSX syntax
migration order and additive fiscal persistence
DTE payload, numbering, provider, retry, cancellation, and invalidation rules
permissions, route guards, tenant isolation, audit, and secret boundaries
endpoint SSRF hardening
CSS, JSON, Compose YAML, imports, documentation, and archive reproducibility
```

## Source baseline supplied by the workstation

The user-provided v0.11 runtime log showed successful API/web builds, TypeScript completion, 31 generated Next.js pages, healthy PostgreSQL/Auth/API/web services, migration 010, and passing authentication, package, public-catalog, Quote Portal, and Billing smoke tests. v0.12 was built as an overlay on that confirmed source baseline.

## Go validation completed in the build environment

The complete API source was copied into an isolated contract harness. External Firebase Admin and pgx implementations were replaced only by local compile-compatible declarations; RentStage packages and tests remained the real source.

```bash
GOTOOLCHAIN=local GOWORK=off go test ./...
GOTOOLCHAIN=local GOWORK=off go vet ./...
```

Result:

```text
PASS
```

DTE tests cover:

- control-number construction and sequence extraction;
- document type selection;
- required issuer/receiver snapshot fields;
- receiver identification/address requirement for Factura totals over $1,095;
- Factura and CCF payload sections and monetary values;
- unsafe MOCK/PRODUCTION and auto-submit settings;
- local MOCK sign/accept/invalidate lifecycle;
- recursive credential redaction for outbound requests and persisted provider responses;
- preservation of the provider/environment/schema identity captured at DTE preparation;
- rejection when the tenant DTE settings change during preparation;
- production HTTPS enforcement;
- private, loopback, link-local, metadata, credentialed, and unsupported endpoint rejection;
- bounded exponential retry delay.

Existing identity, billing, customer, quote, reservation, availability, package, catalog, and portal tests also passed.

## TypeScript validation completed in the build environment

The complete v0.12 `app`, `components`, and `lib` trees were checked using strict framework contracts.

```bash
tsc -p tsconfig.json --pretty false
```

Result:

```text
PASS
```

The TS/TSX syntax pass covered 74 application/configuration source files (plus `next-env.d.ts` in tree accounting), including DTE inbox/detail/settings, invoice-DTE actions, customer receiver fields, billing location codes, navigation, route guards, formatting, and shared API types.

## Migration and persistence checks

Expected embedded order:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
005_calendar_operations.sql
006_identity_access.sql
007_packages.sql
008_public_catalog.sql
009_quote_portal.sql
010_billing_payments.sql
011_dte_integration.sql
```

Static review confirmed that migration 011:

- is additive relative to v0.11;
- adds customer DTE identity/location fields with safe defaults;
- adds issuer/receiver DTE snapshots to invoices;
- backfills existing invoices once before DTE preparation;
- adds DTE item type, unit code, and product code snapshots;
- creates tenant-scoped DTE settings, documents, and events;
- protects invoice links with composite tenant foreign keys;
- enforces supported document/provider/environment/status values;
- makes generation code, control number, and idempotency key unique per tenant;
- prevents more than one active DTE per invoice while allowing replacements after rejection/invalidation/cancellation;
- seeds MOCK settings without transmitting existing data;
- reuses the existing `set_updated_at()` trigger helper;
- contains no `DROP TABLE`, `TRUNCATE`, or business-data deletion.

A real PostgreSQL 18 migration execution remains a workstation integration check.

## Transaction and concurrency review

```text
DTE prepare
  lock invoice
  validate invoice and fiscal state
  lock the complete tenant DTE settings row and sequence
  reject a concurrent settings change
  build immutable payload from the locked settings
  insert DTE + event
  consume control number
  commit

DTE submission start
  lock DTE
  validate state and attempt budget
  mark SUBMITTING
  mark invoice SUBMITTED
  append event
  commit

provider result
  lock SUBMITTING DTE
  persist redacted request/response/seal/errors
  derive ACCEPTED / REJECTED / RETRY_REQUIRED
  update invoice fiscal status
  append event
  commit

DTE cancellation
  lock READY_TO_SIGN / RETRY_REQUIRED DTE
  mark CANCELLED
  restore invoice READY_FOR_DTE
  append event
  commit

DTE invalidation
  lock ACCEPTED DTE
  mark INVALIDATION_PENDING
  call provider
  persist separate invalidation evidence
  derive INVALIDATED or restore ACCEPTED
  update invoice fiscal status when accepted
  append event
  commit
```

## Security review

- Actual credentials are resolved only from `env://...` references.
- Database records and API configuration responses contain references, not secret values.
- Provider request and response evidence recursively redact password/token/secret fields.
- Authentication responses/tokens are not persisted.
- A prepared DTE preserves its provider mode, environment, and schema version for submission and invalidation even if the tenant changes current settings later.
- Production endpoints require HTTPS and TLS 1.2 or newer.
- URL credentials are rejected.
- DNS results and redirects are checked against internal/private/link-local targets.
- HTTP responses are size-limited and requests have explicit timeouts.
- DTE reads and mutations require authenticated session, active membership, server-resolved tenant, and `fiscal.read`/`fiscal.manage`.
- Repositories filter by context tenant and use composite foreign keys.
- Invoice void is blocked while an active prepared/submitted/accepted DTE exists.
- Cancelling preparation does not reuse its control number.
- MOCK cannot be configured as PRODUCTION.

## Formatting and static checks

Completed checks include:

- `gofmt` over Go source;
- Go tests and vet;
- strict TypeScript and TS/TSX syntax transpilation;
- resolution of 235 internal aliased imports across the complete 75-file TypeScript declaration/source tree;
- JSON parsing;
- Docker Compose YAML parsing;
- CSS delimiter balance and DTE selector coverage;
- migration filename/order checks;
- route/permission consistency;
- PowerShell script delimiter and unsafe-character review;
- secret/private-file exclusion review;
- trailing-whitespace and archive-path review.

## Runtime validation required on the workstation

This packaging environment has no Docker Engine, PostgreSQL 18 server, Firebase Authentication emulator, or PowerShell runtime. The definitive integration sequence is:

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a

Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz

powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1 -SkipSubmission
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-quote-portal.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-billing.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-dte.ps1
```

The workstation must also confirm migration 011 against the existing v0.11 PostgreSQL volume and manually inspect the DTE payload/history pages.

## Packaging checks

Before release:

- the update archive contains exactly the 46 files added or changed from v0.11.0;
- `.env`, `.git`, `node_modules`, `.next`, database files, backups, volumes, credentials, and validation harnesses are excluded;
- ZIP CRC integrity passes;
- SHA-256 is generated and verified;
- applying the update archive to a clean v0.11 tree is compared byte-for-byte with the complete v0.12 source tree.

## Boundaries intentionally left open

- MOCK seals are not fiscal seals.
- MH_HTTP has not been independently certified against a live taxpayer's DGII onboarding contract.
- No automatic background submission/reconciliation worker exists.
- No official legible PDF/QR or public-query link is generated.
- No contingency flow, credit/debit notes, purchases, input-tax credit, books, or F-07 preparation exists.
- Secret Manager is used through environment-variable injection, not a direct SDK resolver.
