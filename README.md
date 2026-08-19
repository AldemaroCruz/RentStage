# RentStage Starter v0.15.3

> **Demo comercial para CONAMYPE**: el recorrido conecta inventario, cotización, reserva, facturación y cobro, y ahora incorpora un inbox tipo WhatsApp que recomienda paquetes reales y prepara cotizaciones con aprobación humana.

RentStage is a multi-tenant rental-operations SaaS foundation. It begins with professional audio equipment and is designed to expand into studios, rehearsal rooms, services, public catalogs, fiscal operations, and AI-assisted customer conversations through controlled application APIs.

## New in v0.15.3

- Completes the commercial demo from chat to customer decision: an authorized operator can issue or rotate the existing secure Quote Portal link from the assistant conversation.
- Keeps the link credential out of persistent data. The raw token exists only in the no-store issuance response and the current browser tab's `sessionStorage`; chat and audit records contain sanitized evidence only.
- Shows portal views, acceptance/rejection, and the resulting reservation back in the conversation. Only an explicit customer acceptance in the portal can create a reservation.
- Adds a persistent, accessible light/dark mode button throughout the admin and public experiences, plus a presenter-friendly **Reiniciar demo** action.
- Extends the existing assistant smoke test across secure sharing and explicit rejection without adding Meta credentials, external delivery, a database migration, or cloud infrastructure.

See [`docs/WHATSAPP-CUSTOMER-PORTAL-0.15.3.md`](docs/WHATSAPP-CUSTOMER-PORTAL-0.15.3.md), [`docs/UPGRADE-0.15.3.md`](docs/UPGRADE-0.15.3.md), and [`docs/VALIDATION-0.15.3.md`](docs/VALIDATION-0.15.3.md).

## Quality update in v0.15.2

- Expands frontend unit coverage from 65.27% to 99%+ of executable lines and from 50% to 100% of functions.
- Covers every exported formatter used by inventory, quotes, reservations, warehouse operations, billing, payments, deposits, dates, and currencies.
- Exercises missing Cloud Run identity-token failure and fallback-cache paths without contacting Google metadata services.
- Enforces minimum frontend coverage in the existing `test:ci` command: 95% lines, 90% branches, and 95% functions.
- Changes no application behavior, database schema, infrastructure, staging gate, secret, permission, or external integration.

See [`docs/UPGRADE-0.15.2.md`](docs/UPGRADE-0.15.2.md) and [`docs/VALIDATION-0.15.2.md`](docs/VALIDATION-0.15.2.md).

## New in v0.15.1

- Completes the zero-credential sales-chat demo with a visible **Enviar respuesta demo** action and clear simulated-delivery labels.
- Supports repeated customer follow-ups and creates a fresh deterministic response draft after every inbound demo message; every draft still requires human action.
- Lets an authorized operator select an existing customer or create and link a `WHATSAPP`-sourced customer directly from the conversation.
- Audits customer linking, simulated inbound replies, and simulated outbound delivery while preserving tenant boundaries and the existing RBAC model.
- Keeps the real Meta channel disabled and documents the separate production adapter work required for a paid WhatsApp Business rollout.

See [`docs/WHATSAPP-DEMO-CHAT-0.15.1.md`](docs/WHATSAPP-DEMO-CHAT-0.15.1.md) and [`docs/UPGRADE-0.15.1.md`](docs/UPGRADE-0.15.1.md).

## New in v0.15.0

- Adds an authenticated WhatsApp-style sales inbox that works immediately in `DEMO` mode, without a Meta Business account or phone number.
- Converts a structured customer inquiry into a deterministic, auditable package recommendation using current tenant prices and availability.
- Keeps a human in control: the proposed response is editable and only `OWNER`, `ADMIN`, or `MANAGER` can approve it.
- Creates a quote in `DRAFT` after approval, never sends a real message, never confirms a reservation, and never blocks inventory automatically.
- Adds tenant-scoped conversations, messages, proposal evidence, approval actors, a seeded CONAMYPE scenario, and a provider boundary for a future Meta/Vertex connection.

See [`docs/WHATSAPP-ASSISTANT-0.15.0.md`](docs/WHATSAPP-ASSISTANT-0.15.0.md) for the product and safety boundary and [`docs/UPGRADE-0.15.0.md`](docs/UPGRADE-0.15.0.md) for deployment steps.

## Included from v0.14.2

- Adds a manual **Staging Cost Control** workflow with `status`, `pause`, and `resume` operations.
- Suspends Cloud SQL instance charges while the commercial demo is not in use, without deleting its database, Terraform state, secrets, images, or Cloud Run services.
- Requires the automatic deployment gate to be disabled before a pause and requires explicit confirmation for every state change.
- Stops the main application pipeline before image builds when Cloud SQL is paused and provides an actionable recovery message.

Cloud Run already uses zero minimum instances. The cost-control workflow therefore leaves its stable URLs and Identity Platform configuration intact and targets the staging database, which is the principal fixed runtime cost. See [`docs/STAGING-COST-CONTROL.md`](docs/STAGING-COST-CONTROL.md).

## Fixed in v0.14.1

- Keeps the sidebar inside the dynamic viewport and makes its navigation independently scrollable on short desktop windows and mobile devices.
- Keeps the workspace switcher visible at the bottom while preserving access to every operations, inventory, finance, and system option.
- Adds a subtle cross-browser scrollbar and touch momentum without changing permissions, routes, or application data.

## New in v0.14.0

- Adds an authenticated seven-minute commercial walkthrough with live readiness checks over the real RentStage modules.
- Adds a coherent, idempotent demo storyline: accepted quote, confirmed reservation, issued invoice, and partial bank-transfer payment.
- Links the dashboard and navigation directly to the guided presentation and the tenant's public catalog.
- Makes the product boundary explicit: DTE is safe only as `MOCK / TEST`.
- Adds focused readiness unit tests and release-specific upgrade and validation documentation.

This release adds no schema migration, API endpoint, GCP resource, environment variable, or permission. When demo seeding is enabled, it adds stable commercial records to the demo tenant through the existing seed mechanism.

## Included from v0.13.6

- Keeps the documented local owner credentials available only when the web build explicitly uses the Firebase Authentication emulator.
- Starts non-local login forms empty and never renders the local password in staging HTML.
- Replaces the obsolete v0.5 dashboard milestone with the current product/demo-readiness status and an explicit DTE MOCK / TEST boundary.
- Adds focused runtime-configuration tests so an invalid non-local emulator value fails closed.
- Records the successful first GCP staging deployment and synchronizes release metadata after the v0.13.5 operational fixes.

This release has no database migration, API contract, GCP resource, environment-variable, or tenant-data change. Local Docker Compose behavior and its documented demo account remain unchanged.

## New in v0.13.0

- GitHub Actions pull-request and `main` pipeline covering repository contracts, Go unit/race/vet checks, Node tests, TypeScript, the production Next.js build, Terraform validation, source/dependency security, Docker integration, all six product smoke suites, image scanning, staging deployment, and rollback.
- Separate manual `Staging Infrastructure` workflow for Terraform plan/apply with GCS remote state.
- Scheduled and pull-request CodeQL workflow, plus Dependabot configuration for Go modules, npm, GitHub Actions, and Terraform providers.
- Keyless GitHub → Google Cloud authentication through Workload Identity Federation; no service-account JSON key is required or supported.
- Terraform staging foundation for Artifact Registry, PostgreSQL 18 on Cloud SQL, Secret Manager, Firebase/Identity Platform, runtime service accounts, and least-purpose IAM.
- Public Cloud Run web service and private Cloud Run API service. The web service obtains a Google-signed identity token from the metadata server and forwards it in `X-Serverless-Authorization`.
- Cloud Run-aware configuration validation: secure cookies, HTTPS origins, real Firebase project, no emulator/bootstrap outside local development, explicit anti-abuse salt, and controlled staging demo data.
- Parameterized PowerShell smoke tests that run against either the local Firebase emulator or staging Identity Platform through the public web proxy.
- Deployment hardening: private API IAM, generated Cloud Run hostname registration in Identity Platform, Cloud SQL Unix socket, Secret Manager injection, post-deploy smoke tests, and automatic revision rollback when staging validation fails.
- Local `compose.yaml` remains the development environment and still runs the full authenticated, commercial, billing, and DTE MOCK test suite.

See [`docs/STAGING-CICD.md`](docs/STAGING-CICD.md) for the first staging setup and [`docs/UPGRADE-0.13.0.md`](docs/UPGRADE-0.13.0.md) for the incremental upgrade.

## New in v0.12.1

- Hotfix for DTE result persistence on PostgreSQL: accepted/rejected/invalidation timestamps no longer reuse a `VARCHAR` status parameter inside prepared-statement `CASE` expressions.
- Automatic, local-only recovery of `MOCK / TEST` documents left in `SUBMITTING` by v0.12.0.
- Structured DTE mutation errors in API logs and improved smoke-test diagnostics.

- Separate DTE domain over the v0.11 invoice ledger; quotes, reservations, invoices, payments, deposits, and DTE records remain independent.
- Tenant DTE settings for provider, environment, document type, schema version, establishment, point of sale, endpoints, secret references, sequence, and retry policy.
- Immutable DTE preparation from issued invoice snapshots, with generation code, control number, idempotency key, payload, provider evidence, receipt seal, attempts, errors, and event history.
- Supported local document builders for `01` Factura and `03` Comprobante de Crédito Fiscal.
- Explicit lifecycle: `READY_TO_SIGN`, `SUBMITTING`, `ACCEPTED`, `REJECTED`, `RETRY_REQUIRED`, `INVALIDATION_PENDING`, `INVALIDATED`, and `CANCELLED`.
- Transactional tenant control-number allocation; cancelled, rejected, or invalidated numbers are never reused.
- No-network `MOCK / TEST` provider for repeatable local signing, acceptance, rejection/retry tests, receipt-seal simulation, and invalidation.
- Configurable `MH_HTTP` adapter for official authentication, signing, reception, and invalidation endpoints supplied during taxpayer onboarding.
- Environment-only secret resolution through references such as `env://DTE_MH_PASSWORD`; raw credentials are not stored in PostgreSQL.
- Endpoint hardening against URL credentials, localhost, private/link-local/metadata destinations, unsafe redirects, unbounded bodies, and insecure production transport.
- DTE inbox, detail/evidence page, tenant integration settings, invoice preparation/submission actions, permissions, audit events, and `scripts/smoke-dte.ps1`.
- Incremental migration `011_dte_integration.sql`.

> The `MOCK-...` seal has **no fiscal validity**. `MH_HTTP` is an integration boundary, not a claim of DGII certification. Production use requires the taxpayer's official authorization, current schemas/catalogs, credentials, signer requirements, service endpoints, certification cases, contingency rules, and legible-representation requirements.

## Billing & Payments included from v0.11.0

- Tenant fiscal/billing profile, internal invoices, tax snapshots, receivables, partial payments, reversals, security deposits, and financial dashboard.
- Exact tax-included/tax-exclusive calculations using cent-based monetary rules.
- Issued invoices provide the immutable seller, receiver, line, tax, and amount snapshots consumed by the DTE module.

## Quote Portal included from v0.10.0

- Expiring and revocable customer links for formal quotes.
- Public quote review and acceptance/rejection without a RentStage account.
- Final availability validation and atomic online acceptance → `PENDING` reservation.
- Versioned terms, response evidence, token rotation, privacy-reduced conflicts, and no-store protections.

## Identity foundation included from v0.6.0

- **Next.js 16 + React 19 admin panel** with login, signup, onboarding, workspace selection, team management, organization settings, and profile pages.
- **Go modular-monolith REST API**.
- **PostgreSQL 18** with embedded incremental migrations.
- **Firebase Authentication emulator** for repeatable local email/password testing.
- **Firebase Admin Go SDK** for ID-token exchange, HttpOnly session cookies, and revocation-aware session verification.
- **Role-based access control** for `OWNER`, `ADMIN`, `MANAGER`, and `STAFF`.
- **Membership-validated tenant context** on every protected API request.
- **CSRF protection** for every unsafe API method.
- **Real user identity in audit records** instead of the previous `local-admin` placeholder.
- All v0.1–v0.5 functionality: catalog, physical inventory, customers, quotes, availability, reservations, warehouse operations, calendar, agenda, and alerts.

## Runtime

```text
Browser
  │
  ├── Firebase JS SDK ───────────────→ Authentication Emulator :9099
  │                                      Emulator UI :4000
  │
  ▼
Next.js web :3000
  │ same-origin proxy (/api/backend/*)
  │ forwards HttpOnly cookies + CSRF header
  ▼
Go API :8080
  │
  ├── Firebase Admin SDK ────────────→ Authentication Emulator :9099
  ├── DTE provider
  │     ├── MOCK / TEST              no external network
  │     └── MH_HTTP                  configured official endpoints
  │
  ▼
PostgreSQL :5432 inside Docker
           :5433 on the Windows host by default
```

## Identity vs. authorization

RentStage deliberately separates two responsibilities:

```text
Firebase Authentication
────────────────────────────────
Who is the user?

email/password
identity UID
ID token
session-cookie verification
account disabled/revoked state

PostgreSQL / RentStage
────────────────────────────────
What may the user do?

organizations
memberships
active workspace
role
permissions
tenant isolation
invitations
audit actor
```

The browser never supplies an authoritative tenant ID. A selected workspace is stored in an HttpOnly cookie, but the API accepts it only after finding an active membership for the authenticated user.

## Roles

| Capability | OWNER | ADMIN | MANAGER | STAFF |
|---|---:|---:|---:|---:|
| Read dashboard, calendar, inventory, customers, quotes, reservations | Yes | Yes | Yes | Yes |
| Create/update customers | Yes | Yes | Yes | Yes |
| Read reusable packages | Yes | Yes | Yes | Yes |
| Create/update package composition and pricing | Yes | Yes | No | No |
| Read public-catalog settings | Yes | Yes | Yes | Yes |
| Publish packages/resources and change storefront settings | Yes | Yes | No | No |
| Read web quote requests | Yes | Yes | Yes | Yes |
| Change/convert web quote requests | Yes | Yes | Yes | No |
| Read Quote Portal settings and response evidence | Yes | Yes | Yes | Yes |
| Configure Quote Portal and issue/rotate/revoke links | Yes | Yes | Yes | No |
| Read financial dashboard, invoices, payments, and deposits | Yes | Yes | Yes | Yes |
| Configure fiscal profile and create/issue/void invoices | Yes | Yes | Yes | No |
| Record/void payments and manage security deposits | Yes | Yes | Yes | No |
| Read DTE settings, payloads, status, seals, and evidence | Yes | Yes | Yes | Yes |
| Configure, prepare, submit, retry, cancel, and invalidate DTE | Yes | Yes | Yes | No |
| Create/update quotes | Yes | Yes | Yes | No |
| Create/reprogram/cancel reservations | Yes | Yes | Yes | No |
| Assign, check out, return, and complete equipment | Yes | Yes | Yes | Yes |
| Change catalog and prices | Yes | Yes | No | No |
| Manage physical inventory records | Yes | Yes | Yes | No |
| Read complete audit log | Yes | Yes | No | No |
| Manage organization settings | Yes | Yes | No | No |
| Manage team and invitations | Yes | Yes, limited hierarchy | No | No |

The frontend hides unavailable actions for usability. The Go API independently enforces the same permissions; hiding a button is never treated as a security boundary.

## Requirements

### Local development

- Docker Desktop/Engine with Docker Compose v2.
- Internet access for the first dependency/image build.

### Staging delivery

- GitHub repository with Actions enabled.
- Billing-enabled Google Cloud project.
- `gcloud` and authenticated GitHub CLI for the one-time bootstrap.
- A GitHub Environment named `staging`.

See `docs/STAGING-CICD.md`; do not create a long-lived Google service-account JSON key.

## Start locally

```bash
cp .env.example .env
docker compose config
docker compose up --build -d
docker compose ps -a
```

Open:

- RentStage login: http://127.0.0.1:3000/login
- API health: http://127.0.0.1:8080/healthz
- API readiness: http://127.0.0.1:8080/readyz
- Authentication Emulator UI: http://127.0.0.1:4000
- Authentication Emulator REST API: http://127.0.0.1:9099

### Local owner account

When `LOCAL_AUTH_BOOTSTRAP=true`, the API creates or reconciles this user in the Authentication emulator:

```text
Email:    owner@rentstage.local
Password: RentStage123!
Role:     OWNER
Tenant:   AudioPro Demo
```

These credentials are for local development only. Change them in `.env` or disable bootstrap when they are no longer useful.

### Windows: PostgreSQL port 5432 already in use

The API communicates with PostgreSQL inside Docker at `db:5432`. The published host port is only for local database tools. Keep this in `.env` when Windows already uses port 5432:

```env
POSTGRES_PORT=5433
```

Do not change the internal `DATABASE_URL` in `compose.yaml`.

## Upgrade to v0.13.0

v0.13.0 has no new database migration. Back up PostgreSQL, extract the incremental ZIP over v0.12.1, and rebuild locally:

```powershell
docker compose down
Expand-Archive -Path "$HOME\Downloads\rentstage-upgrade-v0.13.0.zip" -DestinationPath . -Force
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a
powershell -ExecutionPolicy Bypass -File .\scripts\run-smoke-suite.ps1
```

Do not run `docker compose down -v`. Full instructions are in `docs/UPGRADE-0.13.0.md`.

## Historical upgrade from v0.5.0

Extract the v0.6 upgrade package over the existing project and preserve `.env`:

```powershell
docker compose down
docker compose build --no-cache api auth web
docker compose up -d
docker compose ps -a
```

Do **not** use `docker compose down -v`. Migration `006_identity_access.sql` preserves the complete rental dataset and adds identity/access tables and columns.

See `docs/UPGRADE-0.6.0.md` for the complete backup, upgrade, verification, and rollback procedure.

## Authentication flow

```text
1. Browser signs in through Firebase Authentication.
2. Firebase returns a short-lived ID token.
3. Browser requests a RentStage CSRF token.
4. Browser POSTs the ID token + CSRF token to /api/v1/auth/session.
5. Go verifies the ID token and requires recent authentication.
6. Go synchronizes the identity into PostgreSQL.
7. Go creates an HttpOnly Firebase session cookie.
8. The browser clears its temporary Firebase client state.
9. Protected requests verify the session, membership, tenant, role, and permission.
```

## Main API routes

### Session and identity

```text
GET    /api/v1/auth/csrf
POST   /api/v1/auth/session
DELETE /api/v1/auth/session
GET    /api/v1/auth/me
POST   /api/v1/auth/select-tenant
```

### Organizations and team

```text
POST   /api/v1/organizations
GET    /api/v1/tenant
PATCH  /api/v1/tenant
GET    /api/v1/team
POST   /api/v1/team/invitations
DELETE /api/v1/team/invitations/{invitationId}
PATCH  /api/v1/team/members/{userId}
GET    /api/v1/invitations/{token}
POST   /api/v1/invitations/{token}/accept
```

### Packages

```text
GET    /api/v1/packages
POST   /api/v1/packages
GET    /api/v1/packages/{packageId}
PATCH  /api/v1/packages/{packageId}
DELETE /api/v1/packages/{packageId}
GET    /api/v1/packages/{packageId}/quote-template?quantity=1
POST   /api/v1/packages/{packageId}/availability
```

### Public catalog and quote requests

Anonymous storefront routes (the two POST operations still require a valid CSRF cookie/header pair):

```text
GET    /api/v1/public/catalogs/{tenantSlug}
GET    /api/v1/public/catalogs/{tenantSlug}/packages/{packageSlug}
GET    /api/v1/public/catalogs/{tenantSlug}/resources/{resourceSlug}
POST   /api/v1/public/catalogs/{tenantSlug}/availability
POST   /api/v1/public/catalogs/{tenantSlug}/quote-requests
```

Authenticated tenant-administration routes:

```text
GET    /api/v1/public-catalog
PATCH  /api/v1/public-catalog
PATCH  /api/v1/public-catalog/packages/{packageId}
PATCH  /api/v1/public-catalog/resources/{resourceId}
GET    /api/v1/quote-requests
GET    /api/v1/quote-requests/{requestId}
PATCH  /api/v1/quote-requests/{requestId}
POST   /api/v1/quote-requests/{requestId}/convert
```

All authenticated catalog, quote-request administration, commercial, and operational routes require a valid session, active membership, server-resolved tenant context, and the appropriate permission.

### Quote Portal

Anonymous customer routes use the bearer token in `X-RentStage-Quote-Token`. The two decision operations also require the normal anonymous CSRF cookie/header pair:

```text
GET    /api/v1/public/quote-portal
POST   /api/v1/public/quote-portal/accept
POST   /api/v1/public/quote-portal/reject
```

Authenticated tenant routes:

```text
GET    /api/v1/quote-portal-settings
PATCH  /api/v1/quote-portal-settings
POST   /api/v1/quotes/{quoteId}/send
POST   /api/v1/quotes/{quoteId}/portal/reissue
DELETE /api/v1/quotes/{quoteId}/portal
```

The generated browser URL has this shape:

```text
http://127.0.0.1:3000/q#<one-time-bearer-token>
```

The `/q` page immediately moves the fragment token into tab-scoped `sessionStorage`, removes it from the visible address using `history.replaceState`, and sends it only through the dedicated request header. The API persists only its SHA-256 hash.

### Billing & Payments

Authenticated tenant routes:

```text
GET    /api/v1/billing/settings
PATCH  /api/v1/billing/settings
GET    /api/v1/billing/tax-rules
GET    /api/v1/billing/dashboard

GET    /api/v1/invoices
POST   /api/v1/invoices
GET    /api/v1/invoices/{invoiceId}
PATCH  /api/v1/invoices/{invoiceId}
POST   /api/v1/invoices/{invoiceId}/issue
POST   /api/v1/invoices/{invoiceId}/void

GET    /api/v1/payments
POST   /api/v1/payments
GET    /api/v1/payments/{paymentId}
POST   /api/v1/payments/{paymentId}/void

GET    /api/v1/security-deposits
POST   /api/v1/security-deposits
GET    /api/v1/security-deposits/{depositId}
POST   /api/v1/security-deposits/{depositId}/receive
POST   /api/v1/security-deposits/{depositId}/settle
```

Invoice creation accepts `MANUAL`, accepted `QUOTE`, or non-cancelled `RESERVATION` sources. Final invoice numbering is assigned only on issue; payments update receivables transactionally; security deposits never enter invoice revenue or estimated IVA.

### DTE Integration

Authenticated tenant routes:

```text
GET    /api/v1/dte-settings
PATCH  /api/v1/dte-settings
GET    /api/v1/dte
GET    /api/v1/dte/{documentId}
GET    /api/v1/invoices/{invoiceId}/dte
POST   /api/v1/invoices/{invoiceId}/dte
POST   /api/v1/dte/{documentId}/submit
POST   /api/v1/dte/{documentId}/retry
POST   /api/v1/dte/{documentId}/cancel
POST   /api/v1/dte/{documentId}/invalidate
```

`fiscal.read` protects settings/evidence reads. `fiscal.manage` protects configuration and lifecycle mutations. Every repository call derives `tenant_id` from the authenticated membership context.

Preparing a DTE freezes the issued invoice snapshots into a new fiscal record. Submission persists provider request/response evidence and synchronizes the invoice fiscal status. A manual retry reuses the same generation code, control number, payload, and idempotency key.

## Recommended v0.12 walkthrough

1. Complete the tenant and customer fiscal/location fields in **Configuración de facturación** and the customer detail page.
2. Open **Integración DTE**, keep `MOCK / TEST`, and confirm establishment, point-of-sale, schema, sequence, and retry settings.
3. Create and issue an internal invoice with a complete fiscal snapshot.
4. Open the invoice and select **Preparar DTE**.
5. Review the immutable JSON, generation code, control number, document type, and event history.
6. Select **Firmar y transmitir** and confirm `ACCEPTED` plus a `MOCK-...` receipt seal.
7. Open the DTE inbox, inspect redacted provider evidence, and verify the invoice fiscal state.
8. Invalidate the MOCK document and confirm that the original payload, seal, and event history remain preserved.
9. Run all regression smoke tests and `smoke-dte.ps1`.
10. Keep `MH_HTTP / PRODUCTION` disabled until official onboarding and certification are complete.

## Automated local smoke test

After all containers are healthy, run from PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
```

The authentication script signs in through the Authentication emulator, exchanges the ID token for an HttpOnly RentStage session, reads `/auth/me`, calls a tenant-protected dashboard endpoint, and logs out.

To validate the seeded package, quote template, and package availability endpoint, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
```

To validate the anonymous storefront, privacy-reduced availability response, quote-request intake, and authenticated inbox, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1
```

To validate secure token issuance/rotation, anonymous document access, online acceptance, idempotency, reservation creation, and evidence persistence, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-quote-portal.ps1
```

To validate invoice issue, IVA calculation, partial payment, reversal, deposit lifecycle, and the financial dashboard, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-billing.ps1
```

To validate DTE settings, complete fiscal snapshots, preparation, numbering, MOCK signing/reception, immutable evidence, invalidation, and cleanup, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-dte.ps1
```

## Useful commands

```bash
make up
make ps
make logs
make api-logs
make web-logs
make auth-logs
make db-logs
make db-shell
make smoke
make smoke-packages
make smoke-public
make smoke-quote-portal
make smoke-billing
make smoke-dte
make reset
```

## Security boundary

Version 0.13 preserves the complete local v0.12.1 product while adding a staging-only production foundation: GitHub Actions, Terraform, Workload Identity Federation, Artifact Registry, Cloud Run, Cloud SQL, Identity Platform, Secret Manager, security gates, and remote smoke testing. Localhost development still uses the Authentication emulator and PostgreSQL Docker volume. MOCK receipt seals remain test evidence and are **not** fiscally accepted documents. Do not expose ports `9099` or `4000` to the public internet, and do not present a `MOCK-...` seal as fiscally accepted.

Before GCP staging/production:

```text
remove FIREBASE_AUTH_EMULATOR_HOST
use a real Firebase / Identity Platform project
set COOKIE_SECURE=true
set REQUIRE_VERIFIED_EMAIL=true
configure production origins and domains
use Secret Manager / workload identity
inject DTE secrets into runtime environment variables
allowlist only the official fiscal endpoints assigned during onboarding
apply Cloud SQL private connectivity, PITR, backups, and retention
use a random PUBLIC_REQUEST_FINGERPRINT_SALT
add platform/WAF rate limiting, alerts, and security monitoring
complete official taxpayer authorization, certification, schemas, catalogs, and cases
validate legible representation, QR/public query, invalidation, and contingency requirements
```

The `MH_HTTP` adapter blocks unsafe destinations and redacts credential-like evidence, but infrastructure egress controls and official endpoint allowlisting are still required. See `docs/DTE-0.12.0.md` and `docs/SECURITY-0.6.0.md`.

## Documentation

- `docs/WHATSAPP-CUSTOMER-PORTAL-0.15.3.md`
- `docs/UPGRADE-0.15.3.md`
- `docs/VALIDATION-0.15.3.md`
- `docs/UPGRADE-0.14.0.md`
- `docs/VALIDATION-0.14.0.md`
- `docs/UPGRADE-0.13.6.md`
- `docs/VALIDATION-0.13.6.md`
- `docs/STAGING-CICD.md`
- `docs/UPGRADE-0.13.0.md`
- `docs/VALIDATION-0.13.0.md`
- `docs/DTE-0.12.0.md`
- `docs/UPGRADE-0.12.0.md`
- `docs/VALIDATION-0.12.0.md`
- `docs/ARCHITECTURE.md`
- `docs/UPGRADE-0.11.0.md`
- `docs/VALIDATION-0.11.0.md`
- `docs/UPGRADE-0.10.0.md`
- `docs/VALIDATION-0.10.0.md`
- `docs/UPGRADE-0.9.0.md`
- `docs/VALIDATION-0.9.0.md`
- `docs/UPGRADE-0.8.0.md`
- `docs/VALIDATION-0.8.0.md`
- `docs/UPGRADE-0.7.0.md`
- `docs/VALIDATION-0.7.0.md`
- `docs/UPGRADE-0.6.0.md`
- `docs/SECURITY-0.6.0.md`
- `docs/requests.http`

## Next increment

Use the v0.15.3 guided presentation with prospective users and CONAMYPE to validate WhatsApp-style intake, customer creation, multi-turn human-approved conversation, package recommendation, secure quote review, and an explicit customer decision. The next channel increment is a controlled Meta Business pilot with webhook verification, templates, opt-in evidence, a 24-hour service-window policy, delivery status, and human escalation. DTE remains MOCK/TEST until a real authorized taxpayer provides the current official onboarding contract.
