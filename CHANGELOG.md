# Changelog

## 0.13.4 - CI gate alignment and Go security patch

### Fixed

- Synchronizes the repository and web package versions at 0.13.4.
- Upgrades the operational Go toolchain and API builder from 1.26.5 to 1.26.6.
- Replaces the provider-incompatible Firebase deletion_policy argument with a provider-independent Terraform lifecycle prevent_destroy guard.
- Unblocks repository validation, Terraform validation, Govulncheck, and the API binary vulnerability scan.

### Compatibility

- No database migration, API contract, application runtime behavior, tenant data, or environment variable changes.
- Historical validation records keep the Go version used when those releases were originally tested.

## 0.13.3 — Frontend TypeScript test-import compatibility

### Fixed

- Enables TypeScript `allowImportingTsExtensions` for the Next.js no-emit
  type-check configuration.
- Allows the Node-native TypeScript unit tests introduced in v0.13.0 to keep
  explicit `.ts` import specifiers while still passing `tsc --noEmit`.
- Unblocks the `npm run test:ci` sequence before coverage tests and the
  production Next.js build.

### Compatibility

- No application runtime behavior, API contract, database migration, `.env`,
  tenant data, or GCP configuration changes.
- Apply over v0.13.2 and rebuild only the web image first.

## 0.13.2 — Go module synchronization shell fix

### Fixed

- Corrects the Windows Docker helper that failed with `sh: go: not found`
  after pulling the then-pinned Go Alpine builder image.
- Removes the Alpine login shell (`sh -l`) from module synchronization. Alpine's
  `/etc/profile` resets `PATH`, which drops `/usr/local/go/bin` from the official
  Go image environment.
- Invokes `/usr/local/go/bin/go` directly through Docker `--entrypoint`, avoiding
  shell and PATH differences on Windows, Linux, and macOS.
- Adds the missing `scripts/ci/sync-go-modules.sh` companion for Linux and macOS.
- Adds a preflight `go version` check and preserves actionable exit codes.

### Compatibility

- No application-code change, database migration, `.env` change, or data change.
- Apply over v0.13.1, rerun module synchronization, then rebuild API and web.


## 0.13.1 — Build and local smoke-suite hotfix

- Regenerates Go module metadata through a Docker-backed helper so Firebase Admin transitive checksums are committed instead of existing only inside an image layer.
- Runs `go mod tidy` before API image tests, preventing the local Docker build from failing on missing Firebase `go.sum` entries.
- Makes the backend CI job fail with a clear instruction when `go.mod` or `go.sum` is not committed in tidy form.
- Makes the local smoke-suite wrapper use PowerShell 7 when available and fall back to Windows PowerShell 5.1.
- No database migration, runtime business behavior, environment variable, or tenant data changes.


## 0.13.0 — Production Foundation & Staging CI/CD

- Adds GitHub Actions gates for Go unit tests, race detection, coverage, vet,
  frontend type checking/build, dependency audits, CodeQL, Gitleaks, Gosec,
  Govulncheck, Trivy and the complete Docker Compose smoke suite.
- Adds a staging-only Cloud Run deployment workflow authenticated with Google
  Cloud Workload Identity Federation; no service-account JSON key is required.
- Adds a one-time PowerShell bootstrap for Artifact Registry, Cloud SQL,
  Secret Manager, runtime service accounts and GitHub OIDC federation.
- Adds non-destructive post-deployment smoke tests for health, web login,
  public catalog and optional Firebase-authenticated tenant access.
- Keeps development on localhost and keeps the DTE provider in MOCK / TEST.
- Adds optional DATABASE_URL support for Cloud SQL Unix-socket connections.
- Adds repository security invariants, migration checks, Dependabot and
  staging runbooks/rollback instructions.


## 0.12.1 — DTE submission-result hotfix

### Fixed

- Corrected the PostgreSQL prepared-statement parameter inference conflict that could leave a DTE in `SUBMITTING` after the local MOCK provider returned an accepted result.
- Submission and invalidation timestamps now use dedicated boolean parameters instead of reusing the `VARCHAR` status parameter inside `CASE` expressions.
- Provider request and response evidence encoding errors are now handled explicitly instead of being ignored.
- Internal DTE mutation failures now write a structured, secret-free error to the API log together with request and tenant context.
- Added migration `012_dte_submission_result_hotfix.sql`, which safely recovers only `MOCK / TEST` documents left in `SUBMITTING` to `RETRY_REQUIRED` and restores their invoices to `READY_FOR_DTE`. External `MH_HTTP` documents are intentionally not changed automatically.
- Improved `scripts/smoke-dte.ps1` so submission, invalidation, and cleanup errors include the HTTP response and request ID, and a stuck `SUBMITTING` state is reported explicitly.

### Compatibility

- No business data is deleted.
- No environment variables change.
- The hotfix requires rebuilding only the API image; rebuilding the web image is unnecessary.
- Existing v0.12.0 DTE payloads, control numbers, generation codes, and idempotency keys are preserved.

## 0.12.0 — El Salvador DTE Integration Foundation

### Added

- Tenant-scoped DTE settings for provider mode, test/production environment, document type, schema version, establishment, point of sale, service endpoints, secret references, sequence, and retry policy.
- Separate DTE aggregate backed by issued invoice snapshots; fiscal records never rebuild seller, receiver, lines, taxes, or totals from mutable live profiles.
- Local builders for document types `01` Factura and `03` Comprobante de Crédito Fiscal, including identification, issuer, receiver, items, summary, IVA, payment condition, extension, and appendix sections.
- Transactional generation code, control number, and idempotency key allocation; consumed numbers are never reused after cancellation, rejection, or invalidation.
- DTE lifecycle and evidence: `READY_TO_SIGN`, `SUBMITTING`, `ACCEPTED`, `REJECTED`, `RETRY_REQUIRED`, `INVALIDATION_PENDING`, `INVALIDATED`, and `CANCELLED`.
- No-network `MOCK / TEST` provider for deterministic signing simulation, acceptance, receipt-seal simulation, retry/rejection cases, and invalidation without external fiscal calls.
- Configurable `MH_HTTP` provider boundary for authentication, signing, reception, and invalidation endpoints supplied during official taxpayer onboarding.
- Environment-only credential references (`env://...`) and optional Compose variables for DTE username, password, and signing password.
- Persisted immutable payload, signed representation, redacted provider request/response evidence, receipt seal, attempts, timestamps, errors, invalidation evidence, and append-only DTE events.
- Fiscal inbox/detail/settings pages, invoice DTE controls, expanded issuer/receiver location fields, permissions, route guards, audit events, REST examples, documentation, and `scripts/smoke-dte.ps1`.
- Incremental migration `011_dte_integration.sql`.

### DTE invariants

```text
Only issued invoices with complete immutable fiscal snapshots may prepare a DTE
Invoice, DTE, provider evidence, and invalidation evidence remain separate records
Preparing locks the invoice and tenant sequence before consuming a control number
A retry reuses the same generation code, control number, payload, and idempotency key
One active DTE is permitted per invoice
Invoice void is blocked while an active prepared/submitted/accepted DTE exists
Cancelling preparation restores invoice fiscal readiness but does not reuse numbering
Provider credentials are resolved at runtime and never persisted
MOCK is TEST-only and its receipt seal has no fiscal validity
MH_HTTP production activation requires HTTPS and explicit official onboarding data
```

### Security and compatibility

- `fiscal.read` is available to OWNER, ADMIN, MANAGER, and STAFF; `fiscal.manage` is limited to OWNER, ADMIN, and MANAGER.
- Every DTE read/mutation requires authenticated session, active membership, server-resolved tenant context, and permission middleware.
- Configurable outbound endpoints reject URL credentials, unsupported schemes, localhost, loopback, private, link-local, metadata, CGNAT, and benchmark addresses; DNS answers and redirects are revalidated.
- Production requests require HTTPS, TLS 1.2+, bounded redirects, 30-second timeouts, and response-size limits.
- Provider evidence recursively redacts keys containing password, token, or secret; authentication tokens are not persisted.
- Migration 011 is additive and preserves all v0.11 catalog, package, portal, quote, reservation, billing, payment, deposit, identity, and audit records.
- MOCK is not a fiscal service. `MH_HTTP` has not been independently certified against a live taxpayer's current official DGII contract.
- Automatic workers/reconciliation, contingency, certified legible representation/QR, credit/debit notes, purchases/input credit/books/F-07, and direct Secret Manager SDK resolution remain follow-on work.

## 0.11.0 — Billing & Payments Core

### Added

- Tenant-scoped fiscal profile and billing policy: legal/trade name, NIT, NRC, economic activity, fiscal address, contact data, invoice prefix, payment terms, tax-included pricing, and default tax rate.
- Versioned tax-rule records with taxable, exempt, and non-taxable categories; the initial El Salvador tenant seed includes a configurable 13% IVA rule without hard-coding tax calculations across the application.
- Internal invoices sourced from accepted quotes, active reservations, or manual lines, with customer/seller/tax/price snapshots, draft editing, sequential issue numbering, due dates, overdue projection, voiding, and printable documents.
- Exact cent-based line calculations for tax-exclusive and tax-included prices, proportional header-discount allocation, line/invoice range checks, and single-currency enforcement per tenant.
- Payments with partial or full allocations across one or more same-customer invoices, confirmed/voided lifecycle, exact allocation totals, deterministic invoice locking, receivable-status updates, and reversal of allocations when a payment is voided.
- Security deposits separated from sales revenue and IVA, with pending/received/partially settled/returned/retained/settled states and cumulative returned/retained balances.
- Financial dashboard for issued, collected, outstanding, overdue, estimated output tax, deposits held, recent invoices/payments, and six-month billing/collection trends.
- Customer fiscal fields for tax identification, registration number, and billing address.
- `billing.read`, `billing.manage`, `payment.read`, and `payment.manage` permissions; finance navigation, route guards, tenant isolation, and audit events.
- Incremental migration `010_billing_payments.sql`, REST examples, upgrade/validation guides, and `scripts/smoke-billing.ps1`.

### Financial invariants

```text
Quote, reservation, invoice, payment, and DTE remain separate domain records
Draft invoices have no final sequential invoice number
Issuing locks tenant billing settings before consuming the next number
Every invoice line stores price, tax rule, tax rate, tax category, and calculated snapshots
Payment allocations must sum exactly to the payment amount
Allocated invoices must belong to the same tenant, customer, and currency
Concurrent payments lock invoices in deterministic UUID order
Voiding a payment reverses invoice paid balances and status in one transaction
Security deposits do not increase invoice revenue or estimated IVA
Financial dashboard values aggregate only the tenant base currency
Internal invoices are not Hacienda DTEs and are never represented as fiscally accepted
```

### Security and compatibility

- Every billing query uses tenant context resolved from the authenticated membership; request bodies never choose an authoritative tenant.
- OWNER and ADMIN manage billing and payments; MANAGER manages operational billing/payments; STAFF has read-only financial access.
- Invoice issue, payment allocation/reversal, and deposit settlement use PostgreSQL transactions and row locks.
- Monetary values are range-checked for `NUMERIC(14,2)` and calculated through cent integers at the business-rule boundary.
- Migration 010 is additive and preserves v0.10 identity, catalog, package, public-catalog, quote-request, Quote Portal, quote, reservation, warehouse, and audit data.
- Direct Hacienda DTE signing/transmission, fiscal invalidation, notes of credit/debit, purchase books, F-07 preparation, and payment-gateway integrations remain v0.12+ work.

## 0.10.0 — Quote Portal

### Added

- Tenant-scoped Quote Portal settings for customer copy, accent color, default validity, rejection policy, required response name, acceptance terms text, and terms version.
- Expiring public quote portals generated when an administrator sends a `DRAFT` quote.
- One-time raw bearer link returned only on send/reissue; PostgreSQL stores only a SHA-256 token hash.
- Fragment-to-header browser transport at `/q`, keeping the secret out of URL paths, query strings, server request logs, referrers, and indexable markup.
- Tenant-branded public quote document with customer, event, period, resource lines, prices, totals, contact details, terms, and response state.
- Customer acceptance and rejection with optional email/reason, response-name policy, evidence timestamps, origin fingerprint, User-Agent, decision source, and idempotency.
- Transactional online acceptance that locks the quote, recalculates availability, creates a `PENDING` reservation with historical quote lines, and marks the quote/portal accepted in one commit.
- Privacy-reduced public availability-conflict responses without internal IDs, physical assets, serials, or raw capacity counts.
- Portal revision, rotation, revocation, global enable/disable behavior, expiration, view count, first-view event, and synchronization with administrator quote decisions.
- Quote-detail portal controls and evidence panel, plus `/settings/quote-portal` administration.
- Public/no-store response headers and CSP/referrer/indexing protections for the `/q` surface.
- Incremental migration `009_quote_portal.sql`, API examples, and `scripts/smoke-quote-portal.ps1`.

### Quote Portal invariants

```text
Raw token is generated from 256 random bits and never persisted
Browser link keeps the token in the URL fragment, then removes it from the address
Public API receives the token only through X-RentStage-Quote-Token
Rotating the portal invalidates the previous token immediately
Global portal disable prevents every public view and decision
Only SENT quotes can be accepted or rejected by a customer
Online acceptance rechecks availability before changing commercial state
Reservation creation + quote acceptance + portal evidence commit atomically
Availability conflict leaves quote and portal active for staff adjustment
Repeated customer decisions are idempotent
Terms text/version are immutable snapshots per issued portal revision
Every protected admin action derives tenant_id from authenticated membership context
```

### Security and compatibility

- `quote.read` permits settings/evidence reads; `quote.manage` permits configuration, send, rotation, and revocation.
- Anonymous decision endpoints still require the existing CSRF cookie/header pair.
- Public responses use `no-store`, `no-referrer`, `noindex`, frame denial, and content-type protections.
- Raw client IP addresses are not persisted; evidence stores a salted SHA-256 origin fingerprint.
- Migration 009 is additive and preserves all v0.9 catalog, package, quote-request, customer, quote, reservation, identity, and audit data.
- Transactional email delivery, notifications, reminders, payments, WhatsApp, and AI orchestration remain follow-on work.

## 0.9.0 — Public Catalog

### Added

- Tenant-branded anonymous storefront at `/p/[tenantSlug]` with public package and resource detail pages.
- Authenticated public-catalog administration for storefront headline, description, cover image, accent color, contact data, terms, price/resource visibility, and quote-request enablement.
- Per-package and per-resource publication controls with featured ordering, resource public slugs, public descriptions, and public images.
- Anonymous package availability endpoint that delegates to the existing reservation-aware engine and returns a privacy-reduced result without resource IDs, serialized assets, or internal capacity counts.
- Anonymous quote-request flow with package quantities, event period, contact data, consent, preliminary availability, and estimated-price snapshots.
- Authenticated quote-request inbox, detail view, status workflow, and transactional conversion into a customer plus `DRAFT` quote.
- Historical request snapshots for packages, quote lines, prices, currency, availability, terms text, and terms version.
- Tenant-scoped anti-abuse fingerprinting, five-requests-per-hour origin limit, honeypot behavior, strict input validation, and CSRF protection for anonymous writes.
- `public_catalog.read`, `public_catalog.manage`, `quote_request.read`, and `quote_request.manage` permissions.
- Public-catalog and quote-request audit events, including an explicit API actor for anonymous submissions.
- Incremental migration `008_public_catalog.sql`, demo storefront seed, REST examples, and `scripts/smoke-public-catalog.ps1`.

### Public-catalog invariants

```text
Disabled catalogs are indistinguishable from missing catalogs
Only active, ready, explicitly published packages are anonymous
Only active, explicitly published resources with tenant-unique slugs are anonymous
Public availability never exposes physical asset identifiers or raw capacity counts
Anonymous requests do not block inventory and do not create reservations
Request package, price, availability, currency, and terms snapshots remain historical
Conversion creates/reuses a same-tenant customer and creates a DRAFT quote atomically
Every administrative read and mutation derives tenant_id from authenticated context
```

### Security and compatibility

- OWNER and ADMIN can configure publication and manage requests; MANAGER can read the storefront configuration and manage requests; STAFF has read-only access to catalog configuration and requests.
- Anonymous unsafe endpoints still require the existing CSRF cookie/header pair.
- Raw client IP addresses are not stored; the rate-limit key is a salted SHA-256 tenant-scoped fingerprint.
- `PUBLIC_REQUEST_FINGERPRINT_SALT` must be an explicit random value of at least 32 characters outside local development.
- Migration 008 is additive and preserves all v0.8 tenants, packages, resources, customers, quotes, reservations, warehouse history, sessions, memberships, and audit events.
- Online acceptance/rejection of a formal quote remains deferred to v0.10.0.

## 0.8.0 — Packages Core

### Added

- Tenant-scoped reusable commercial packages with names, slugs, descriptions, suggested guest capacity, optional images, lifecycle status, and ordered resource composition.
- Package items with resource quantities, customer-facing descriptions, optional unit-price overrides, and deterministic ordering.
- Two pricing strategies: `SUM_ITEMS` and `FIXED`, with calculated component value, effective selling price, commercial discount, and surcharge read models.
- Package list, create, detail, update, archive/reactivation, read-only inspection, responsive cards, and availability panels in the admin UI.
- Package availability checks for a blocked interval and package quantity through the existing reservation-aware availability engine.
- Quote-template expansion that converts a package into ordinary quote lines with historical descriptions, quantities, prices, and allocated discounts.
- Package selector in the quote editor with availability feedback before inserting one or more package quantities.
- `package.read` and `package.manage` permissions with navigation filtering, route guards, and API enforcement.
- Package audit events: `PACKAGE_CREATED`, `PACKAGE_UPDATED`, and `PACKAGE_ARCHIVED`.
- Incremental migration `007_packages.sql` with composite tenant foreign keys, pricing checks, indexes, and update triggers.
- Demo **Paquete Fiesta 100 personas**, including speakers, subwoofers, mixer, microphones, and a new cable kit resource.
- Package-oriented REST examples and `scripts/smoke-packages.ps1`.

### Package invariants

```text
Package belongs to exactly one tenant
Package items may reference only active resources from the same tenant
Each resource appears at most once per package
Fixed-price packages preserve their exact commercial total in quote snapshots
Package changes never mutate quotes already created from the package
Availability multiplies every component quantity by the requested package count
Physical asset assignment remains a reservation-preparation responsibility
```

### Security and compatibility

- OWNER and ADMIN may manage packages; MANAGER and STAFF receive read access.
- Package repositories derive `tenant_id` from authenticated server context; request bodies cannot select another organization.
- Migration 007 is additive and preserves all v0.7 catalog, customer, quote, reservation, warehouse, identity, session, and audit data.
- Public catalog, anonymous quote requests, tenant storefronts, and customer-side online acceptance remain intentionally deferred.

## 0.7.0 — Team & Access UI refinement

### Changed

- Rebuilt the **Equipo y permisos** section headers with explicit spacing, borders, and typography instead of the previously undefined `panel-heading` style.
- Aligned member-table headers, values, status badges, activity text, and action buttons on the same grid.
- Increased member and invitation text sizes for better readability without changing the overall RentStage visual language.
- Added responsive member cards below 1180 px and a stacked mobile layout below 700 px, removing the fragile horizontal table overflow.
- Added clear labels for role, status, and activity in responsive layouts.
- Corrected singular/plural rendering (`1 miembro`, `2 miembros`).
- Localized membership and invitation status labels in Spanish.
- Added explicit button types and table semantics for safer form behavior and improved accessibility.

### Compatibility

- Frontend-only release.
- No database migration.
- No Go API or HTTP contract change.
- No Firebase Authentication, server-session, CSRF, RBAC, or tenant-isolation change.
- Existing PostgreSQL and Firebase emulator volumes are preserved.

## 0.6.2 — Next.js prerender Suspense fix

### Fixed

- Wrapped the root route/search-parameter consumer in a React `Suspense` boundary.
- Fixed the production `next build` failure while prerendering `/_not-found` / `/404`.
- The same boundary covers the login and signup pages, which also consume `useSearchParams()` for safe post-authentication redirects.
- Added a branded initialization fallback for the suspended client subtree.

### Compatibility

- No database migration.
- No Go API or HTTP contract change.
- No authentication, authorization, tenant-isolation, or persisted-volume change.
- Existing PostgreSQL and Firebase emulator data are preserved.

## 0.6.1 — Auth emulator Java 21 build fix

### Fixed

- Changed the local authentication emulator base image from `node:24-bookworm-slim` to `node:24-trixie-slim`.
- OpenJDK 21 is now installed from Debian 13 stable repositories, fixing the Docker build failure where Bookworm could not resolve `openjdk-21-jre-headless`.
- Added build-time checks for `java -version` and `firebase --version`.

### Compatibility

- No database migration.
- No API contract change.
- Existing PostgreSQL and Firebase emulator volumes are preserved.
- All v0.6.0 identity, membership, role, CSRF, and tenant-isolation behavior is unchanged.

## 0.6.0 — Identity & SaaS Foundation

### Added

- Firebase Authentication emulator service and persistent local auth export.
- Firebase JS SDK login/signup flow with in-memory browser persistence.
- Firebase Admin Go integration for ID-token verification and server session cookies.
- HttpOnly session and workspace cookies.
- CSRF bootstrap and validation for every unsafe API request.
- Authenticated user synchronization into PostgreSQL.
- Multi-workspace onboarding and workspace switching.
- `OWNER`, `ADMIN`, `MANAGER`, and `STAFF` roles with explicit permissions.
- Membership hierarchy rules in addition to endpoint permissions.
- Organization, team, invitation, and profile settings pages.
- Hashed, expiring, one-time membership invitations.
- Active/suspended membership lifecycle.
- Route-level and control-level permission guards in the admin panel.
- Real user identity in audit read models and mutation actors.
- Local PowerShell authentication smoke test.
- Incremental migration `006_identity_access.sql`.

### Authentication invariants

```text
Firebase proves identity
RentStage proves membership and authorization
ID token exchange requires recent authentication
Browser tenant selection is validated against ACTIVE memberships
Unsafe requests require CSRF cookie + header equality
Every business endpoint requires session + tenant + permission
```

### Local-only components

- The included Firebase Authentication emulator is for local development and must not be exposed publicly.
- `LOCAL_AUTH_BOOTSTRAP` and the demo credentials must be disabled outside local development.
- Production must use a real Firebase Authentication / Identity Platform project and secure cookies.

### Intentionally deferred

- GCP staging and production infrastructure.
- Production email verification and password-reset UX.
- MFA and federated login.
- Invitation email delivery.
- Subscription billing and plan enforcement.
- Public catalog, AI agent, and WhatsApp integration.

## 0.5.0 — Calendar & Operations Center

### Added

- Month, week, and daily agenda views based on blocked reservation periods.
- Calendar filters for status, customer, resource, and free-text search.
- Daily operations read model for departures, event starts, expected returns, and returned reservations awaiting closure.
- Operational alerts for overdue returns, preparation not started, incomplete physical assignment, delayed check-out, and pending completion.
- Dashboard metrics for today's departures and returns, overdue equipment, active reservation value, and active reservations.
- Manual reservation creation with historical prices and transactional availability validation.
- Reservation source attribution: `QUOTE`, `MANUAL`, `WEB`, `WHATSAPP`, and `AI_AGENT`.
- Safe reservation rescheduling that excludes itself from availability and checks already assigned physical units.
- Immutable `reservation_schedule_history`.
- Schedule history UI and structured availability/physical-asset conflict responses.
- Incremental migration `005_calendar_operations.sql`.

### Scheduling invariants

```text
Manual create   validates customer + resources + availability in one transaction
Reschedule      allowed only before CHECKED_OUT
Availability    excludes the reservation being moved
Assigned assets must remain conflict-free in the new period
Every change    writes schedule history + audit event
```

### Intentionally deferred

- Authentication and role-based access.
- Public catalog and customer self-service booking.
- Package recommendations.
- QR scanning.
- Payments, WhatsApp, LLM orchestration, and subscription billing.

## 0.4.0 — Warehouse Operations

### Added

- Exact physical-asset assignment for reservation items while preparing an order.
- Warehouse read model with required, assigned, missing, and eligible units.
- Transactional asset assignment with row locking and overlapping-reservation conflict detection.
- Manual unassignment before check-out.
- Inventory-completeness enforcement before `PREPARING → READY`.
- Reservation and per-asset check-out timestamps, actors, and notes.
- Return inspection for every checked-out asset.
- Automatic asset status updates for good, maintenance-required, damaged, and lost returns.
- Atomic release of assignments when a pre-check-out reservation is cancelled.
- Warehouse activity history separate from reservation status history.
- Structured HTTP conflicts for incomplete preparation, double assignment, and return mismatch.
- Incremental migration `004_warehouse_operations.sql`.
- Warehouse panel, check-out modal, return-inspection UI, API examples, and upgrade documentation.

### Warehouse invariants

```text
Assign/unassign       only in PREPARING
Mark ready            requires complete exact assignment
Check out             only from READY
Return                must inspect every checked-out asset exactly once
Complete              only after RETURNED with no outstanding units
Cancel                releases active assignments before check-out
```

### Return-condition mapping

```text
GOOD                 → AVAILABLE
MAINTENANCE_REQUIRED → MAINTENANCE
DAMAGED              → DAMAGED
LOST                 → LOST
```

### Intentionally deferred

- QR label generation and mobile scanning.
- Partial or staged returns.
- Warehouse-aware protection around manual asset-condition editing.
- Calendar and delivery routing.
- Authentication, public catalog, payments, WhatsApp, and LLM orchestration.

## 0.3.0 — Availability and Booking Cores

### Added

- Bulk temporal availability endpoint for one or many resources.
- Availability read model with requested, eligible, reserved, available, and fulfillable quantities.
- Transaction-scoped advisory resource locks in deterministic order.
- Transactional accepted-quote conversion into a reservation.
- Reservation monetary snapshots copied from accepted quote items.
- Reservation list, search, filtering, detail document, and quote/customer navigation.
- Reservation operational states: pending, confirmed, preparing, ready, checked out, returned, completed, and cancelled.
- Explicit transition endpoints and immutable reservation status history.
- Availability-conflict responses that identify every affected resource.
- Quote read models linked to their generated reservation.
- Booking Core dashboard messaging and reservation navigation.
- Incremental migration `003_booking_core.sql`.
- API examples, upgrade instructions, architecture notes, and validation documentation.

### Availability behavior

The following reservation states block inventory:

```text
PENDING | CONFIRMED | PREPARING | READY | CHECKED_OUT
```

The following states release it:

```text
RETURNED | COMPLETED | CANCELLED
```

### Intentionally deferred

- Creating standalone/manual reservations from the UI.
- Assigning physical asset IDs to reservation items.
- QR scanning and warehouse checklists.
- Calendar UI and delivery routing.
- Payments, deposits, and online acceptance.
- Authentication and production tenant authorization.
- WhatsApp, LLM orchestration, and human handoff.

## 0.2.0 — Customer and Quote Cores

### Added

- Customer list, search, source filtering, create, edit, profile, and commercial history.
- E.164 phone normalization and email validation.
- Quote list, filters, draft editor, historical line prices, totals, detail document, and status badges.
- Draft → sent → accepted/rejected lifecycle plus cancellation.
- Transactional persistence for quote headers and items.
- Customer and quote audit events.
- Incremental migration `002_customer_quotes.sql` with indexes and monetary integrity constraints.
- Demo customers and quotes.
- API examples and upgrade documentation.

### Intentionally deferred

- Quote delivery through email or WhatsApp.
- Availability blocking from quotes.
- Quote-to-reservation conversion.
- Authentication and production tenant authorization.
