# RentStage v0.11.0 validation record

## Scope

This record covers the incremental v0.10.0 → v0.11.0 Billing & Payments Core update.

Validated areas:

```text
Go compilation contracts, tests, and vet
TypeScript strict contracts and TS/TSX syntax
migration order and additive persistence design
invoice/tax/payment/deposit invariants
permissions, route guards, tenant isolation, and audit registration
CSS, JSON, Compose YAML, imports, documentation, and archive reproducibility
```

## Source baseline supplied by the workstation

The user-provided v0.10 runtime log showed successful API/web builds, Next.js TypeScript and route generation, healthy PostgreSQL/Auth/API/web services, migration 009, and the Quote Portal smoke flow. v0.11 was built as an overlay on that source baseline.

## Go validation completed in the build environment

The full API source was copied into an isolated contract harness. External Firebase Admin and pgx implementations were replaced only by local compile-compatible declarations; RentStage packages and tests remained the real source.

Commands:

```bash
GOTOOLCHAIN=local GOWORK=off go test ./...
GOTOOLCHAIN=local GOWORK=off go vet ./...
```

Result:

```text
PASS
```

Billing tests cover:

- tax-exclusive 13% line calculation;
- tax-included 13% extraction;
- exempt calculation;
- exact proportional header-discount allocation;
- monetary database-range rejection;
- aggregate invoice totals;
- exact payment-allocation validation;
- default invoice-prefix normalization.

## TypeScript validation completed in the build environment

The complete v0.11 `app`, `components`, `lib`, and Next configuration trees were checked using strict TypeScript framework contracts.

```bash
npx tsc -p tsconfig.json --pretty false
```

Result:

```text
PASS
```

The TS/TSX syntax pass covered 71 source files, including billing dashboard, invoices, printable invoice, payments, deposits, billing settings, customer fiscal fields, quote/reservation invoice links, navigation, route guards, formatting, and shared API types.

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
```

Static review confirmed that migration 010:

- is additive relative to v0.10 business tables;
- adds customer tax/billing fields with non-null defaults;
- creates tenant-scoped settings, tax, invoice, payment, allocation, and deposit tables;
- uses existing composite tenant foreign keys where business records are linked;
- preserves price, seller, customer, and tax snapshots on invoices;
- assigns final invoice numbers under a row lock on tenant billing settings;
- uses generated balance columns plus monetary/status constraints;
- prevents more than one active invoice per quote or reservation;
- reuses the existing `set_updated_at()` trigger function;
- seeds settings/rules without issuing or altering commercial records;
- contains no `DROP TABLE`, `TRUNCATE`, or business-data deletion.

A real PostgreSQL 18 migration execution remains a workstation integration check.

## Transaction and concurrency review

Static review confirmed:

```text
invoice issue
  lock invoice
  lock tenant billing settings
  consume tenant next_invoice_number
  write snapshot + event
  commit

payment create
  sort invoice UUIDs
  lock every invoice deterministically
  validate tenant/customer/currency/state/balance
  insert payment + allocations
  update receivables + invoice events
  commit

payment void
  lock payment
  lock allocated invoices deterministically
  reverse paid balances/statuses
  mark payment VOIDED + events
  commit

security-deposit settlement
  lock deposit
  validate cumulative returned/retained totals
  derive lifecycle state
  update balances
  commit
```

## Financial invariants reviewed

- Quotes, reservations, invoices, payments, deposits, and future DTE records remain separate entities.
- Only accepted quotes and non-cancelled reservations may seed invoices.
- Draft invoices are editable and have no final number.
- Issued/paid invoices are immutable except for payment state and controlled voiding.
- Issued invoices cannot be voided while a confirmed allocation remains applied.
- Payment allocations must equal the recorded payment amount.
- Every allocation belongs to the same tenant, customer, and currency and cannot exceed the invoice balance.
- Voiding a payment restores invoice paid amounts and status in the same transaction.
- Deposits remain outside invoice revenue and tax aggregates.
- Dashboard financial totals use the tenant base currency.
- `READY_FOR_DTE` means only that required profile fields were present when issued; it is not Hacienda acceptance.

## Authorization and isolation checks

New permissions:

```text
billing.read
billing.manage
payment.read
payment.manage
```

Role mapping:

```text
OWNER    manage billing and payments
ADMIN    manage billing and payments
MANAGER  manage billing and payments
STAFF    read billing and payments
```

Every billing endpoint is registered through authenticated session, active membership, server-resolved tenant context, and permission middleware. Repositories use the context tenant in all reads/mutations and composite foreign keys prevent cross-tenant links.

Route-guard and navigation checks covered:

```text
/billing
/invoices
/invoices/new
/invoices/[id]
/invoices/[id]/print
/payments
/payments/[id]
/security-deposits
/settings/billing
```

## Formatting and static checks

Completed checks include:

- `gofmt` over Go source;
- strict TypeScript pass and TS/TSX syntax transpilation;
- internal aliased-import resolution;
- JSON parsing;
- Docker Compose YAML parsing;
- CSS delimiter balance and selector coverage for new pages;
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
```

The workstation must also confirm migration 010 against the existing v0.10 PostgreSQL volume and manually exercise invoice print rendering.

## Packaging checks

Before release:

- the update archive contains only files added or changed from v0.10.0;
- `.env`, `.git`, `node_modules`, `.next`, database files, backups, volumes, credentials, and validation harnesses are excluded;
- ZIP CRC integrity passes;
- SHA-256 is generated and verified;
- applying the update archive to a clean v0.10 tree is compared byte-for-byte with the v0.11 source tree.

## Boundaries intentionally left open

- No Hacienda authentication, signer, DTE JSON generation/transmission, reception seal, contingency, invalidation, credit/debit note, or fiscal PDF/QR exists in v0.11.
- Tax rules are seeded/read by the core; a full tax-rule administration UI and statutory catalog synchronization remain future work.
- Internal invoice print output is browser-rendered HTML, not a certified fiscal representation.
- Purchase invoices, supplier expenses, input-tax credit, F-07 preparation, bank reconciliation, online payment gateways, refunds, and multi-currency accounting remain future increments.
