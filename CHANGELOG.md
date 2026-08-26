# Changelog

## 0.20.0 — AI-assisted conversation drafts

### Added

- Adds a web-chat draft-provider boundary with deterministic rules and an optional Vertex AI adapter using Application Default Credentials.
- Adds validated AI mode, project, location, model, timeout, and output-token configuration with safe local and staging defaults.
- Adds provenance metadata and reviewer-facing badges for Vertex AI, deterministic rules, and safe fallback drafts.
- Adds Terraform and deployment wiring for the Vertex AI API and the API runtime service account.

### Reliability and human control

- Routes both initial and follow-up web-chat drafts through the configured provider while preserving client-message idempotency.
- Falls back to deterministic, length-bounded copy when the primary provider fails or returns an invalid draft.
- Keeps generated drafts private and requires an authenticated user to review and explicitly publish every response.
- Performs no automatic external delivery, quote approval, reservation, or inventory mutation.

### Build and compatibility

- Adds and enforces `package-lock.json` through local, Docker, CI, security-audit, and version-consistency workflows.
- Validates the Vertex output-token bound before its `int32` conversion and covers both supported limits plus platform-sized overflow input.
- Upgrades the Alpine OpenSSL runtime libraries used by every web image stage to the patched repository versions during the build.
- Keeps migration ordering at `016_omnichannel_web_chat.sql`; no database migration or public chat API change is required.
- Keeps `rules` as the default runtime mode so existing installations need no cloud credential or Vertex AI access.

## 0.19.1 — Resilient web-chat recovery and polling

### Fixed

- Reuses the same client message identifier when a visitor retries unchanged content, allowing the existing API idempotency contract to prevent duplicate inbound messages and assistant drafts.
- Preserves a valid tab-scoped chat session when restoration fails because of a temporary browser or network error; only missing or expired sessions clear the stored credential.
- Replaces overlapping interval polling with sequential, cancellable polling that pauses while the page is hidden and resumes immediately when the visitor returns.
- Adds bounded polling backoff at 4, 8, 16, and 30 seconds, a visible reconnection notice, localized transport failures, and immediate synchronization when the widget opens.
- Clarifies that the seven-day server session remains resumable while the visitor keeps the browser tab.

### Security and compatibility

- Raw web-chat tokens remain exclusively in tab-scoped `sessionStorage`; PostgreSQL continues to store only SHA-256 hashes.
- Human approval remains mandatory, assistant drafts remain private, and retry behavior creates no automatic quote, reservation, inventory, Meta, or cloud-delivery side effect.
- Adds no migration, environment variable, secret, IAM permission, Terraform resource, WebSocket, service worker, or external provider.

## 0.19.0 — Omnichannel core and first-party web chat

### Added

- Adds an optional chat widget to every public-catalog route, with a responsive visitor form, resumable browser-tab session, message history, short polling, and tenant accent styling.
- Adds tenant-scoped public endpoints to create a conversation, retrieve its visible messages, and add idempotent inbound messages.
- Adds seven-day web-chat sessions, SHA-256 token hashes, consent and terms evidence, a honeypot, a 2,000-character message limit, and a 60-message-per-hour session limit.
- Routes `WEB_CHAT` into the existing assistant inbox with explicit channel labels, visitor email context, response drafts, and authenticated human publication.
- Adds migration `016_omnichannel_web_chat.sql` and an independent `web_chat_enabled` public-catalog setting.

### Human-control and privacy boundary

- Assistant drafts are never returned by the public session API. A visitor sees an outbound message only after an authorized user explicitly publishes it.
- The raw 256-bit session token is returned once, kept only in the visitor tab's `sessionStorage`, sent through a dedicated header, and stored server-side only as a hash.
- Public chat responses are non-cacheable, invalid credentials are indistinguishable from missing sessions, and disabling either the catalog or web chat makes the channel unavailable.
- Web chat introduces no third-party messaging service, WebSocket infrastructure, production secret, IAM grant, Terraform resource, real Meta delivery, automatic quote approval, or inventory reservation.

## 0.18.1 — Meta application readiness and consent safety

### Added

- Adds public privacy, terms, data-deletion, and support routes with an explicit placeholder warning until `NEXT_PUBLIC_SUPPORT_EMAIL` is configured.
- Adds deterministic Spanish/English opt-in and opt-out classification, an authenticated secret-free readiness endpoint, and a tenant-scoped WhatsApp template catalog foundation.
- Extends message delivery states with `DELIVERED` and `READ`, preserving monotonic provider updates.

### Safety boundary

- Opted-out contacts are closed without an automated draft, and subsequent human sends are rejected until an explicit opt-in is received.
- `META_OUTBOUND_ENABLED=true` is accepted only by the loopback local harness. Cloud delivery is rejected in v0.18.1 even when credentials are present.
- No real access token, phone number, webhook subscription, production infrastructure apply, or production application deployment is introduced.

## 0.18.0 — Local Meta WhatsApp contract harness

### Added

- Adds local GET verification and signed POST webhook endpoints using the WhatsApp Business webhook envelope. HMAC SHA-256 is checked against the exact request body before any payload parsing.
- Adds tenant-scoped public channel connections, inbound message and delivery-status normalization, idempotent external message IDs, a 24-hour customer-service window, and human-review drafts in the existing assistant inbox.
- Adds a Graph-compatible text-send client and an isolated loopback implementation that returns provider-style `wamid.local.*` identifiers without contacting Meta or a real telephone.
- Adds a PowerShell smoke test covering raw challenge verification, signed inbound delivery, duplicate redelivery, and the local Graph send contract.
- Adds focused Go tests for webhook verification/signatures, parsing, delivery failure, configuration isolation, and Graph request shape.

### Security and deployment boundary

- `local_mock` is accepted only with `APP_ENV=local` and an HTTP loopback URL ending in the private local Graph harness path; cloud mode requires an HTTPS Graph origin and complete credentials.
- Access tokens, app secrets, and verify tokens are never stored in PostgreSQL. The new migration contains only tenant association and public sender identifiers.
- Staging and production keep Meta disabled because their workflows inject none of the local variables. Existing production secret containers remain empty.
- The production infrastructure apply reviewed in v0.17.1 is deliberately deferred. Both production gates remain false, no production infrastructure or application is applied, and no Meta account or sender is required for this release.

## 0.17.1 — Protected production infrastructure apply

### Added

- Adds a manual production infrastructure apply workflow behind `main`, the protected `production` GitHub Environment, an explicit repository gate, the exact production project ID, and the `APPLY-PRODUCTION` confirmation phrase.
- Generates a saved Terraform plan and applies that same file within one ephemeral job; the plan is never uploaded because it contains the database password.
- Adds an offline JSON safety gate that rejects updates, replacements, destroys, root-level resources, and resources targeting any project other than the protected production project.
- Adds a dedicated apply service account and separate Workload Identity Pool/provider whose OIDC condition accepts only the production apply workflow on `main` in the `production` Environment.
- Adds just-in-time `grant`, `status`, and `revoke` operations for the apply identity's control-plane roles. Bootstrap grants it state-bucket access but no persistent project mutation role.
- Adds focused tests for allowed create-only plans and rejected update, replacement, destroy, root-resource, and cross-project cases.

### Least privilege and safety

- Production API runtime now receives a custom Firebase role containing only `firebaseauth.users.get` and `firebaseauth.users.createSession`; staging keeps its deployed binding so its migration plan remains zero-change.
- The existing production plan identity remains read-only, and the reserved application deployment identity still has no project role or WIF impersonation binding.
- Both `PRODUCTION_INFRA_APPLY_ENABLED` and `PRODUCTION_DEPLOY_ENABLED` remain false by default. This release applies no infrastructure automatically and still contains no production application deployment path.
- Cloud SQL deletion protection, backups, PITR, TLS, connector enforcement, separate state, and Meta secret-value exclusion remain unchanged.

## 0.17.0 — Isolated production infrastructure foundation

### Added

- Adds a reusable Terraform platform module and separate staging and production roots so environments can share reviewed infrastructure code without sharing projects, state, identities, secrets, or lifecycle controls.
- Adds production bootstrap automation for a dedicated billing-enabled GCP project, versioned private state bucket, environment-bound Workload Identity Federation provider, and distinct infrastructure/deployment service accounts without service-account keys.
- Adds a manual **Production Infrastructure Plan** workflow that runs only from `main` through the protected `production` GitHub Environment and intentionally contains no apply or application-deployment path.
- Keeps the production planning identity read-only outside its isolated state bucket and leaves the reserved deployment identity without project roles or GitHub impersonation until a later reviewed apply/deploy increment.
- Adds empty Secret Manager containers for the future Meta access token, app identifiers, app secret, sender/WABA identifiers, and webhook verification token; Terraform never receives their values.
- Adds CI validation for both Terraform roots and offline isolation contracts for state prefixes, GitHub Environment secrets, Meta container scope, and staging state migration declarations.

### Migration and safety

- Preserves the deployed staging names and moves existing Terraform state addresses into `module.platform`; the first staging plan must show moves with zero additions, replacements, or destructions.
- Keeps the production database deletion-protected with TLS, connector enforcement, backups, PITR, and a cost-controlled ZONAL default that must be reviewed before the first apply.
- Does not apply production infrastructure, deploy Cloud Run, seed demo data, create a public sender, contact Meta, or enable a production deployment gate.
- Adds no application API, schema, session, tenant-data, or demo behavior change.

## 0.16.0 — Tenant-scoped operational metrics

### Added

- Adds an authenticated commercial-metrics API and admin workspace with explicit 7, 30, and 90-day reporting windows.
- Measures inquiry volume, quote decisions, quote-derived reservation conversion, first-response time, current pipeline, accepted and reserved value, invoicing, collections, and current receivables.
- Adds six-month value trends, reservation outcomes, customer-source distribution, human-approved assistant messages, Quote Portal decisions, and audit-event evidence.
- Adds a **Métricas** navigation entry and a dashboard shortcut under the existing `operations.read` permission.
- Extends the authenticated integration smoke test and adds focused formatter coverage for metric bars, response durations, and source labels.

### Measurement boundary

- Every query derives the tenant from the authenticated server context; the browser cannot select another tenant ID.
- Windowed activity is distinct from current snapshots. Pipeline and outstanding balances are marked as present-time values.
- Funnel stages summarize activity within one window and are not presented as a closed cohort; manual reservations do not inflate quote-to-reservation conversion.
- No third-party analytics service, tracking cookie, database migration, secret, environment variable, IAM grant, or Terraform resource is introduced.

## 0.15.5 — Administrative dark-theme completion

### Changed

- Completes dark-mode ownership for DTE status banners, public-catalog publication editors, Quote Portal controls, billing actions, and audit timeline events.
- Replaces the last fixed white gradients and cards with shared surface, border, and semantic state tokens while preserving the existing light theme.
- Restores readable secondary copy inside published catalog rows and keeps DTE MOCK and MH_HTTP modes visually distinct without high-contrast white panels.
- Adds subtle themed hover and selected states to the remaining administrative surfaces without changing their layout or behavior.

### Quality and compatibility

- Expands CSS regression coverage for the six remaining administrative screens and their shared control families.
- Keeps appearance persistence, assistant flows, quote decisions, billing, DTE behavior, audit evidence, API contracts, database schema, infrastructure, and CI/CD unchanged.
- Requires no migration, Terraform apply, secret, IAM permission, environment variable, data reset, or manual service operation.

## 0.15.4 — Dark-theme surface and control polish

### Changed

- Replaces remaining fixed white admin surfaces with the shared dark-theme tokens across dashboard metrics, calendars, packages, quotes, reservations, customers, requests, warehouse controls, and dialogs.
- Restores a single aligned visual boundary for search and prefixed inputs instead of rendering a second dark field inside a light container.
- Defines the shared purple, muted-ink, and soft-shadow aliases already consumed by late calendar and dashboard components.
- Preserves semantic green, amber, red, and blue states while reducing harsh contrast and keeping selected, hover, and attention states distinguishable.

### Quality and compatibility

- Adds regression coverage for critical dark surfaces, compound field transparency, and light print tokens.
- Keeps the existing appearance preference, Quote Portal security model, assistant behavior, API contracts, database schema, cloud infrastructure, and CI/CD topology unchanged.
- Requires no migration, Terraform apply, secret, IAM permission, service restart procedure, or data reset.

## 0.15.3 — Customer quote decision and appearance controls

### Added

- Connects the human-approved assistant demo to the existing secure Quote Portal so an operator can issue or rotate a customer link directly from the conversation.
- Shows portal views, decision status, and any resulting reservation in the assistant while preserving the customer's explicit accept/reject action as the only online decision point.
- Adds a persistent light/dark appearance button to authenticated, public, login, onboarding, and Quote Portal screens without a flash of the wrong theme during hydration.
- Adds a presenter reset action that starts a fresh demo inquiry without deleting the immutable history of previous conversations.
- Extends the assistant smoke test through secure link issuance, token-fragment transport, sanitized transcript evidence, anonymous portal review, explicit rejection, and the absence of an automatic reservation.

### Security and compatibility

- The raw 256-bit bearer token is returned only in the no-store issuance response, retained only in browser `sessionStorage`, transported from `/q#token` through the existing request header, and never written to chat messages, audit metadata, or PostgreSQL.
- Sharing requires both `assistant.manage` and `quote.manage`; all reads continue to derive the tenant from the authenticated server context.
- The demo channel still contacts no real telephone, and no Meta credential, webhook, template, cloud resource, database migration, environment variable, or IAM grant is introduced.
- Existing quotes, reservations, portals, conversations, sessions, staging infrastructure, and local data remain compatible.

## 0.15.2 — Frontend coverage gate

### Added

- Covers every exported currency, date, numbering, status, source, warehouse, invoice, payment, and deposit formatter with meaningful known-value, fallback, empty, and invalid-input cases.
- Covers malformed Cloud Run identity tokens, invalid audience URLs, empty metadata responses, and the opaque-token fallback cache without making external requests.
- Enforces minimum frontend coverage of 95% lines, 90% branches, and 95% functions through the existing `npm run test:coverage` and `test:ci` path.

### Quality and compatibility

- Raises measured frontend coverage from 65.27% lines / 83.75% branches / 50.00% functions to at least 99% lines / 96% branches / 100% functions on the current suite.
- Adds no runtime dependency and changes no application behavior, API contract, database schema, infrastructure, staging gate, IAM permission, secret, or tenant data.
- Preserves the single automatic RentStage CI/CD workflow and its parallel jobs.

## 0.15.1 — Interactive WhatsApp demo conversation

### Added

- Adds explicit simulated delivery for human-reviewed outbound messages, without sending anything to a real phone number.
- Adds repeatable inbound demo follow-ups and deterministic assistant drafts so a presenter can demonstrate a complete multi-turn conversation.
- Adds tenant-scoped customer linking and an inline create-and-link flow that reuses the existing customer API, validation, `WHATSAPP` source, and audit events.
- Adds API and smoke coverage for customer linking, first response delivery, customer follow-up, second human-reviewed draft, and follow-up delivery.

### Safety and compatibility

- Real `WHATSAPP` conversations remain blocked from the demo send/receive endpoints; Meta credentials, sender registration, webhooks, templates, and production delivery are intentionally deferred to a separate provider adapter.
- Every generated response remains a `DRAFT` until an authorized owner, admin, or manager explicitly delivers it in the simulator.
- Quote approval still creates only a quote in `DRAFT`; no message, reservation, payment, or inventory block occurs automatically.
- No database migration, GCP resource, environment variable, IAM grant, or real messaging charge is introduced.

## 0.15.0 — Human-approved WhatsApp sales assistant

### Added

- Adds a tenant-scoped WhatsApp-style inbox with conversations, messages, proposal evidence, channel/consent metadata, approval actors, and quote links.
- Adds a zero-credential `DEMO` channel that uses live tenant packages, prices, and availability to produce a deterministic recommendation and editable response.
- Adds owner/admin/manager approval that creates only a quote in `DRAFT`; staff can inspect conversations but cannot approve proposals.
- Adds an idempotent CONAMYPE seed conversation, focused recommendation tests, and product, upgrade, and validation documentation.
- Adds a provider boundary for future Meta WhatsApp Business and Vertex Gemini adapters without coupling them to quote or inventory mutation.

### Safety and compatibility

- No real WhatsApp message is sent in v0.15.0 and no Meta account, phone number, token, secret, or new cloud resource is required.
- The assistant never confirms reservations or blocks inventory. Package availability is checked again at human approval time.
- Adds database migration `013_whatsapp_sales_assistant.sql`, four authenticated API endpoints, and `assistant.read` / `assistant.manage` permissions.
- Existing tenant records are preserved. The seeded conversation is inserted only when the existing demo-data switch is enabled.

## 0.14.2 — Staging cost control

### Added

- Adds a protected manual GitHub Actions workflow to inspect, pause, and resume the staging Cloud SQL instance through the existing keyless infrastructure identity.
- Adds an auditable command-line helper and GitHub job summary for the current Cloud SQL policy, database state, Cloud Run services, and deployment gate.
- Adds offline regression coverage for status, confirmation guards, deployment-gate guards, pause, resume, and unsupported operations.
- Adds an operational runbook with GitHub UI and CLI procedures, expected residual costs, recovery, and Terraform-drift guidance.

### Changed

- Fails the staging deployment before building or pushing images when Cloud SQL is paused.
- Keeps Cloud Run deployed with zero minimum instances so its stable URLs and Identity Platform configuration do not need to be recreated.

### Compatibility

- No database migration, API contract, GCP resource, IAM permission, tenant-data, or application-runtime change.
- Uses the existing `staging` Environment, Workload Identity Federation provider, infrastructure service account, and `rentstage-staging-change` concurrency group.
- Pausing suspends database-backed staging functionality but preserves data. Cloud SQL storage, backups, networking, Artifact Registry, Secret Manager, and other retained resources may still incur charges.

## 0.14.1 — Scrollable application navigation

### Fixed

- Constrains the sidebar to the dynamic viewport so long navigation menus no longer extend below the visible window.
- Makes the navigation region independently scrollable with mouse, keyboard, and touch while keeping the brand and workspace switcher visible.
- Adds a subtle cross-browser scrollbar and prevents scroll chaining from the menu into the page.

### Compatibility

- No database, API, infrastructure, permission, route, or seed change.
- Existing sessions and staging data remain unchanged.

## 0.14.0 — Guided commercial demo

### Added

- Adds a seven-minute authenticated demo route that verifies and presents the live inventory → quote → reservation → invoice → payment journey.
- Adds a coherent idempotent seed scenario for Eventos Marea: an accepted $299 package quote, confirmed reservation, issued invoice, and $150 partial payment.
- Adds guided navigation, presenter talking points, public-catalog access, commercial metrics, and focused readiness unit coverage.

### Changed

- Makes the dashboard entry point and release strip lead directly to the commercial walkthrough.
- Presents WhatsApp + AI as a controlled roadmap differentiator and preserves DTE as an explicit MOCK / TEST boundary.
- Cleans generated TypeScript and accidentally pasted PowerShell content from `.gitignore` hygiene.
- Synchronizes `VERSION`, the web package version, README title, and release documentation at 0.14.0.

### Compatibility

- No database schema migration, API endpoint, GCP resource, environment variable, or permission change.
- With `SEED_DEMO_DATA=true`, the existing seed runner inserts stable demo-tenant commercial records; production environments with seeding disabled are unchanged.
- Seeded records are idempotent and survive an application-image rollback. Use a disposable demo tenant or reset the local database volume when a completely clean dataset is required.

## 0.13.6 — Demo readiness and environment-safe login

### Changed

- Shows the documented owner account and prefills its credentials only in web builds that explicitly use the local Firebase Authentication emulator.
- Starts staging and other non-local login forms empty and omits the local password from their rendered page.
- Replaces the obsolete v0.5 dashboard milestone with the current multi-tenant product status and an explicit DTE MOCK / TEST boundary.
- Synchronizes `VERSION`, the web package version, README title, and release documentation at 0.13.6.
- Adds unit coverage for local/non-local authentication-emulator configuration, including fail-closed handling of invalid values.

### Compatibility

- No database migration, API contract, GCP resource, environment variable, business behavior, or tenant-data change.
- Local Docker Compose continues to default to the Authentication emulator and the documented local owner account.
- Staging continues to build with `NEXT_PUBLIC_USE_AUTH_EMULATOR=false` and therefore exposes no local credentials.

## 0.13.5 - Dependency security patch

### Fixed

- Aligns VERSION, the web package version, and the README title at 0.13.5.
- Upgrades google.golang.org/grpc to v1.82.1.
- Upgrades golang.org/x/net to v0.56.0.
- Upgrades golang.org/x/text to v0.39.0.
- Resolves the reachable Go vulnerability findings and the API binary Trivy findings.

### Compatibility

- No database migration, API contract, runtime behavior, or environment variable changes.

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
