# Upgrade RentStage v0.10.0 → v0.11.0

## Scope

RentStage v0.11.0 introduces **Billing & Payments Core** as an additive update containing only files added or changed since v0.10.0.

```text
tenant billing/fiscal profile
internal invoices and tax snapshots
partial/full payment allocations
security deposits separated from revenue
accounts receivable and financial dashboard
printable internal invoice document
```

The update preserves `.env`, PostgreSQL data, Firebase Authentication emulator data, tenants, memberships, catalogs, packages, quote requests, Quote Portal records, customers, quotes, reservations, warehouse history, and audit history.

> RentStage v0.11 invoices are internal commercial/accounting records. They are not Hacienda DTEs. Signing, schema validation, transmission, reception seals, contingency, invalidation, and fiscal notes remain v0.12 work.

## New migration

```text
010_billing_payments.sql
```

It adds customer billing fields and creates:

```text
billing_settings
tax_rules
invoices
invoice_items
invoice_events
payments
payment_allocations
security_deposits
```

The migration is additive. It seeds tenant billing settings plus initial `IVA`, `EXEMPT`, and `NON_TAXABLE` rules without issuing invoices or changing existing commercial records.

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
  -f /tmp/rentstage-v0.10.0-before-v0.11.0.dump

docker compose cp `
  db:/tmp/rentstage-v0.10.0-before-v0.11.0.dump `
  .\backups\rentstage-v0.10.0-before-v0.11.0.dump

Get-Item .\backups\rentstage-v0.10.0-before-v0.11.0.dump
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

## 3. Apply the update archive

Extract the incremental archive directly into the existing project root:

```powershell
Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.11.0.zip" `
  -DestinationPath . `
  -Force
```

Confirm the release:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.11.0
```

The archive does not contain `.env`; no new environment variable is required for this release.

## 4. Build API and web

Billing changes affect PostgreSQL, Go, and Next.js:

```powershell
docker compose build --no-cache api web

docker compose up -d

docker compose ps -a
```

The API migration runner applies `010_billing_payments.sql` during startup.

## 5. Verify health and migration

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
  -c "SELECT version, applied_at FROM schema_migrations WHERE version = '010_billing_payments.sql';"
```

Confirm the new tables:

```powershell
docker compose exec db `
  psql `
  -U rentstage `
  -d rentstage `
  -c "\dt billing_settings" `
  -c "\dt tax_rules" `
  -c "\dt invoices" `
  -c "\dt invoice_items" `
  -c "\dt invoice_events" `
  -c "\dt payments" `
  -c "\dt payment_allocations" `
  -c "\dt security_deposits"
```

## 6. Run regression and billing smoke tests

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1 -SkipSubmission
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-quote-portal.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-billing.ps1
```

The billing test validates:

```text
billing settings and permissions
active tenant tax rules
manual invoice draft
invoice issue and number assignment
IVA calculation
partial payment allocation
PARTIALLY_PAID transition
payment reversal and receivable restoration
invoice void cleanup
security-deposit receive/return lifecycle when a reservation exists
financial dashboard response
```

## 7. Manual acceptance

### Complete the billing profile

Open:

```text
http://127.0.0.1:3000/settings/billing
```

Configure the legal name, NIT, NRC, economic activity, fiscal address, invoice prefix, payment terms, tax behavior, email, and phone. A complete profile changes newly issued invoices from `NOT_READY` to `READY_FOR_DTE`; this is readiness metadata only and does not transmit anything to Hacienda.

### Create and issue an invoice

Open:

```text
http://127.0.0.1:3000/invoices/new
```

Create an invoice from:

```text
an accepted quote
an active reservation
a manual set of lines
```

Confirm that the draft shows `BORRADOR`, that source lines become immutable price/tax snapshots, and that the final sequential number is assigned only when **Emitir factura** is selected.

### Record a partial payment

Open:

```text
http://127.0.0.1:3000/payments
```

Apply less than the invoice balance and confirm:

```text
invoice status  PARTIALLY_PAID
paid amount     increases by allocation
balance due     decreases by allocation
```

Reverse the test payment and confirm the paid amount, balance, and invoice status are restored atomically.

### Manage a guarantee deposit

Open:

```text
http://127.0.0.1:3000/security-deposits
```

Create a deposit against a reservation, mark it received, then record returned and/or retained cumulative amounts. Confirm the deposit does not alter invoice revenue, paid amount, or estimated IVA.

### Review finance

Open:

```text
http://127.0.0.1:3000/billing
```

Review issued, collected, outstanding, overdue, tax-output estimate, deposits held, and six-month billing/collection trends. Open an invoice and use its print route to review the internal document.

## Rollback

### Application-only rollback

If migration 010 succeeded but the application must temporarily return to v0.10, restore the previous source tree and images without deleting the database. v0.10 ignores the additive tables and customer columns.

### Complete rollback

To remove all v0.11 data and schema changes, restore the pre-upgrade dump into a clean database volume. Do not manually delete billing tables from a production-like database because invoice, payment, allocation, and deposit records are relational and audited.
