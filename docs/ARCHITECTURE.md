# RentStage v0.12 architecture

## Purpose of this release

Versions 0.1–0.11 established the rental, identity, packages, public catalog, Quote Portal, and financial ledger. Version 0.12 adds a distinct fiscal-document boundary over issued invoice snapshots:

```text
Issued internal invoice
        ↓ prepare under invoice + tenant sequence locks
Immutable DTE payload
        ↓ explicit administrative submit
Provider boundary
  ├── MOCK / TEST         local lifecycle validation
  └── MH_HTTP             official onboarding adapter
        ↓
Receipt/rejection evidence
        ↓
DTE and invoice fiscal-state synchronization
```

The application remains a modular monolith. The DTE module reuses authenticated tenant context, permissions, CSRF, invoice snapshots, PostgreSQL transactions, and audit actors while isolating fiscal numbering, payloads, provider evidence, retries, and invalidation from quotes, reservations, receivables, payments, and deposits.

`MOCK` is deliberately non-fiscal. `MH_HTTP` is configurable because production schemas, catalogs, endpoints, credentials, signer requirements, certification cases, and contingency rules must come from the taxpayer's current official onboarding process.

## Local runtime

```text
┌────────────────────────────────────────────────────────────────────┐
│ Browser                                                            │
│                                                                    │
│  Firebase JS SDK ───────────────────────────────────────────┐       │
│  Next.js UI / same-origin API calls                        │       │
└─────────────────────────────────────────────────────────────┼───────┘
                                                              │
                    email/password + ID token                  │
                                                              ▼
                                              ┌────────────────────────┐
                                              │ Firebase Auth Emulator │
                                              │ :9099                  │
                                              │ Emulator UI :4000      │
                                              └───────────▲────────────┘
                                                          │
                                                          │ Admin SDK
                                                          │ user/session APIs
┌────────────────────────┐        HTTP + cookies          │
│ Next.js web :3000      │ ───────────────────────────────┼───────┐
│                        │                                │       │
│ /api/backend/* proxy   │                                │       ▼
└────────────────────────┘                                │  ┌──────────────────┐
                                                          └──│ Go API :8080     │
                                                             │                  │
                                                             │ auth middleware  │
                                                             │ tenant context   │
                                                             │ permission guard │
                                                             └────────┬─────────┘
                                                                      │
                                                                      ▼
                                                             ┌──────────────────┐
                                                             │ PostgreSQL 18    │
                                                             │ source of truth  │
                                                             └──────────────────┘
```

Published local ports:

```text
Web                         3000
API                         8080
PostgreSQL host port        5433
Authentication emulator     9099
Emulator Suite UI           4000
```

Inside the Docker network, PostgreSQL remains `db:5432` and Authentication remains `auth:9099`.

## Deployment shape

RentStage remains:

```text
one Next.js deployable
one Go API deployable
one PostgreSQL transaction boundary
one external identity provider
one optional fiscal provider boundary
```

This gives us secure identity without giving up atomic quote conversion, inventory reservation, physical assignment, return inspection, and membership changes.

## Responsibility split

### Firebase Authentication

Firebase establishes identity only:

```text
UID
email
email verification
password authentication
account disabled state
ID token
session-cookie signature and revocation
```

### RentStage PostgreSQL

RentStage controls authorization and business ownership:

```text
User profile mirror
Organizations / tenants
Memberships
Membership status
Role
Permissions
Selected workspace
Invitations
Business data
Audit history
```

A Firebase project tenant is not used as the RentStage organization model. RentStage organizations continue to be represented by the existing `tenants` table.

## Server-side session flow

```text
Browser                    Go API                    Firebase Auth
   │                          │                           │
   │ email/password           │                           │
   ├─────────────────────────────────────────────────────►│
   │                          │                           │
   │ ID token                │                           │
   ◄─────────────────────────────────────────────────────┤
   │                          │                           │
   │ GET /auth/csrf           │                           │
   ├─────────────────────────►│                           │
   │ CSRF cookie + token      │                           │
   ◄─────────────────────────┤                           │
   │                          │                           │
   │ POST /auth/session       │ verify ID token           │
   │ ID token + CSRF          ├──────────────────────────►│
   │                          │ create session cookie     │
   │                          ◄───────────────────────────┤
   │ HttpOnly session cookie  │                           │
   ◄─────────────────────────┤                           │
   │                          │                           │
   │ Firebase client signout  │                           │
   │                          │                           │
   │ protected request        │ verify session cookie     │
   ├─────────────────────────►├──────────────────────────►│
   │                          │ load membership           │
   │                          │ enforce permission        │
   │ response                 │                           │
   ◄─────────────────────────┤                           │
```

The ID token must represent authentication performed within the previous five minutes before the API exchanges it for a server session.

## Cookies

### `rentstage_session`

```text
HttpOnly   true
Secure     false locally; true in cloud
SameSite   Lax
Path       /
Duration   configurable, 5 minutes–14 days
```

### `rentstage_tenant`

Stores the selected organization ID. It is a preference, not an authorization assertion.

```text
HttpOnly   true
Secure     follows COOKIE_SECURE
SameSite   Lax
```

The tenant middleware validates that the authenticated user has an active membership before accepting the selected workspace.

### `rentstage_csrf`

```text
HttpOnly   true
SameSite   Strict
Lifetime   2 hours
```

The matching token is also returned in the CSRF bootstrap response and must be sent in `X-CSRF-Token` for every unsafe API request.

## Request authorization pipeline

Protected tenant endpoints use this order:

```text
HTTP request
    │
    ▼
request ID
    │
    ▼
CSRF validation for POST/PATCH/PUT/DELETE
    │
    ▼
Firebase session-cookie verification
    │
    ▼
RentStage user status
    │
    ▼
active membership lookup
    │
    ▼
tenant context
    │
    ▼
role → permission check
    │
    ▼
domain handler/service/repository
```

Context available to handlers:

```text
user_id
identity_uid
user_email
user_name
actor_id
tenant_id
role
request_id
```

The previous development headers `X-Tenant-ID` and `X-Actor-ID` are no longer trusted or required.

## Identity data model

```text
Firebase user
     │ identity_uid
     ▼
users
     │
     ├──────────── user_preferences
     │                 └── active_tenant_id
     │
     └── tenant_memberships ───────── tenants
                  │
                  ├── role
                  ├── status
                  ├── invited_by
                  └── joined_at

users ── invited_by ── tenant_invitations ── tenants
                         │
                         ├── email
                         ├── role
                         ├── token_hash
                         ├── status
                         └── expires_at
```

### `users`

The user table mirrors the identity fields needed by the application:

```text
identity_uid
email
display_name
avatar_url
email_verified
status
last_login_at
```

The raw password and Firebase session material are never stored in PostgreSQL.

### `tenant_memberships`

```text
role    OWNER | ADMIN | MANAGER | STAFF
status  INVITED | ACTIVE | SUSPENDED | REMOVED
```

Only `ACTIVE` memberships to `ACTIVE` tenants are returned as workspaces.

### `tenant_invitations`

Invitation tokens are random 256-bit values. Only a SHA-256 hash is persisted.

```text
raw token      returned once in accept_url
SHA-256 hash   stored in PostgreSQL
expiration     seven days
status         PENDING | ACCEPTED | REVOKED | EXPIRED
```

Only one unexpired `PENDING` invitation per `(tenant, email)` is allowed.

## Identity synchronization

When a server session is created:

```text
1. Verify Firebase ID token and revocation state.
2. Load the Firebase user record.
3. Reject disabled or, when configured, unverified users.
4. Locate RentStage user by identity UID or normalized email.
5. Create or update the RentStage user mirror.
6. Ensure user_preferences exists.
7. Load active workspaces and permissions.
8. Issue the server session cookie.
```

Matching by email allows the seeded local owner row to be linked to the emulator UID without changing the existing demo tenant or business data.

## Local owner bootstrap

When enabled, API startup ensures that the Authentication emulator contains:

```text
UID       rentstage-local-owner
Email     owner@rentstage.local
Password  configured by LOCAL_OWNER_PASSWORD
Verified  true
```

The SQL demo seed separately ensures an `OWNER` membership in `AudioPro Demo`. Identity synchronization joins those two records by normalized email.

Bootstrap must be disabled outside local development.

## Role and permission model

Permissions are explicit strings evaluated in the Go API:

```text
tenant.read
tenant.manage
team.manage
audit.read
catalog.read
catalog.manage
package.read
package.manage
inventory.read
inventory.manage
customer.read
customer.manage
quote.read
quote.manage
reservation.read
reservation.manage
warehouse.operate
operations.read
```

### OWNER

Receives every permission and can create organizations, manage team hierarchy, and operate the full rental system.

### ADMIN

Receives every current operational permission, but membership hierarchy adds an additional constraint: an administrator may only manage `MANAGER` and `STAFF` memberships and may not invite or promote another administrator.

### MANAGER

Can read packages and manage customers, quotes, reservations, physical inventory, and warehouse operations. It cannot change package composition or pricing, organization settings, team access, catalog pricing, or the full audit log.

### STAFF

Can read packages and other commercial/operational data, maintain customers, and perform warehouse operations. It cannot modify package pricing, catalog, reservations, organization settings, team access, or audit history.

## Defense in depth

Authorization exists in three places for different reasons:

```text
Navigation filtering   reduces clutter
Route guard            prevents unauthorized UI pages
API permission guard   authoritative security boundary
```

A direct API request cannot bypass the Go permission middleware.

Membership hierarchy is also checked inside the identity service, so the broad `team.manage` permission alone does not allow an ADMIN to modify an OWNER or another ADMIN.

## Tenant isolation

Every protected business repository receives `tenant_id` from server context, not from a request body.

```text
Authenticated user
       │
       ▼
List ACTIVE memberships
       │
       ▼
Validate tenant cookie selection
       │
       ▼
Set server tenant context
       │
       ▼
Repository query WHERE tenant_id = context tenant
```

Existing composite tenant foreign keys continue to prevent cross-organization relationships between categories, resources, assets, packages, package items, customers, quotes, reservations, and assignments.

Recommended regression case:

```text
User belongs only to Tenant B
Customer UUID belongs to Tenant A
GET /customers/{tenant-a-customer-id}
Expected: 404, without disclosing that the UUID exists
```

## Suspension behavior

A membership is checked for every tenant-scoped request. If it changes from `ACTIVE` to `SUSPENDED`, the user's next request cannot establish that workspace context even though the Firebase session cookie itself remains cryptographically valid.

A globally disabled RentStage user or revoked/disabled Firebase account is rejected during session verification.

## Audit actors

Mutations now receive a real actor from authenticated context:

```text
actor_id      RentStage user UUID
actor_name    display name through audit read join
actor_email   user email through audit read join
```

Identity events include:

```text
TENANT_CREATED
TENANT_UPDATED
MEMBERSHIP_INVITATION_CREATED
MEMBERSHIP_INVITATION_ACCEPTED
MEMBERSHIP_INVITATION_REVOKED
MEMBERSHIP_UPDATED
```

All quote, reservation, inventory, warehouse, identity, and package mutations record the authenticated user.

## Backend modules

```text
internal/
├── authn          Firebase integration, session cookies, CSRF, auth handlers
├── core/
│   ├── identity  organizations, memberships, roles, invitations
│   ├── tenant
│   ├── catalog
│   ├── packages  reusable commercial composition and quote templates
│   ├── publiccatalog  storefront publication and quote-request workflow
│   ├── quoteportal    expiring public quotes, decisions, and acceptance orchestration
│   ├── inventory
│   ├── customer
│   ├── quote
│   ├── availability
│   ├── reservation
│   ├── operations
│   ├── dashboard
│   └── audit
├── httpapi        routing and middleware chain
├── config
├── database
└── webutil        request context and JSON contracts
```

## Frontend architecture

```text
RootFrame
  └── AuthProvider
       ├── session probe (/auth/me)
       ├── login/signup through Firebase JS SDK
       ├── ID-token exchange
       ├── workspace selection
       ├── permission set
       └── logout

RootFrame route guards
  ├── public routes (`/login`, `/signup`, `/invites/*`, `/p/*`, `/q`)
  ├── authenticated standalone routes
  ├── onboarding redirect
  └── required permission by route

PublicCatalogShell
  ├── tenant branding and contact information
  ├── published packages/resources
  ├── anonymous availability form
  └── anonymous quote-request form

QuotePortalPage (`/q`)
  ├── fragment bearer capture + address cleanup
  ├── tenant-branded formal quote document
  ├── versioned terms and response form
  └── acceptance/rejection evidence and result

AppShell
  ├── permission-filtered navigation
  ├── workspace switcher
  ├── user/profile menu
  └── operational alerts
```

The Firebase browser state uses in-memory persistence. After the ID token is exchanged, the client signs out from Firebase and relies on the server-managed HttpOnly session.

## Next.js API proxy

The browser calls:

```text
/api/backend/api/v1/...
```

Next.js forwards to the internal Go API while preserving:

```text
Cookie
Content-Type
X-CSRF-Token
X-Request-ID
X-RentStage-Quote-Token
Origin
User-Agent
X-Forwarded-For / X-Real-IP
Set-Cookie responses
Retry-After / Cache-Control / Content-Language responses
```

This keeps API calls same-origin from the browser and prevents the internal Docker hostname from reaching the client.

## Migration 006

`006_identity_access.sql`:

- adds organization logo/address fields;
- links users to Firebase identity UIDs;
- adds email verification and last-login metadata;
- adds membership status, invitation source, joined time, and update time;
- creates `tenant_invitations`;
- creates `user_preferences`;
- adds uniqueness and lookup indexes;
- preserves all previous business records.

## Packages Core

Packages are tenant-scoped commercial templates. They reference catalog resources and quantities, not exact physical asset IDs:

```text
Package: Fiesta 100 personas
│
├── 2 × JBL PRX815W
├── 2 × QSC KS118
├── 1 × Behringer X32 Compact
├── 2 × Shure SM58
└── 1 × Kit de cableado básico
```

This preserves the existing separation between commercial capacity and physical fulfillment:

```text
Package / quote item               reservation asset assignment
resource + quantity                exact serialized unit
2 × Shure SM58                     MIC-SM58-001, MIC-SM58-004
```

### Persistence

```text
tenants
  └── packages
        └── package_items ───────→ resources
```

`packages` stores identity, presentation, lifecycle, pricing mode, and optional fixed price. `package_items` stores ordered resource composition, customer-facing descriptions, quantities, and optional unit-price overrides. Composite `(tenant_id, id)` foreign keys prevent cross-tenant package/resource relationships.

### Pricing

```text
SUM_ITEMS
  effective price = Σ(quantity × current resource/package override price)

FIXED
  calculated price = Σ(quantity × current resource/package override price)
  effective price  = package.fixed_price
```

A fixed price below the component value is allocated proportionally as quote-line discounts. A fixed price above the component value becomes quote-level extra charges. Cent-based allocation ensures that expanded quote lines plus adjustments equal the package commercial total exactly.

### Quote expansion

Packages are templates, not persistent quote dependencies:

```text
package definition
      │
      ▼
GET /packages/{id}/quote-template?quantity=N
      │
      ▼
ordinary quote items with description, quantity, unit price, discount
      │
      ▼
quote snapshots remain unchanged when the package changes later
```

The quote and reservation domains therefore retain their existing transaction boundaries. A later package edit cannot retroactively alter a saved quote or reservation.

Package detail and quote-template reads load the calculated summary and ordered item composition inside one read-only `REPEATABLE READ` transaction. This prevents a concurrent package or catalog-price update from producing a response whose commercial total and item lines come from different database snapshots.

### Availability

```text
POST /packages/{id}/availability
  start_at
  end_at
  quantity
```

The package service multiplies every package-item quantity by the requested package quantity and delegates to the existing reservation-aware availability service. Quotes may still be drafted after a conflict warning because quotes do not block inventory; reservation conversion remains the authoritative availability boundary.

### Authorization

```text
package.read     OWNER | ADMIN | MANAGER | STAFF
package.manage   OWNER | ADMIN
```

All package administration routes are authenticated, membership-resolved, tenant-scoped, and permission-guarded. Version 0.9 exposes only packages that are active, ready, and explicitly marked `public_visible`; anonymous reads never receive internal package or resource UUIDs.

## Migration 007

`007_packages.sql`:

- creates `packages` and `package_items`;
- adds tenant/slug and tenant/resource uniqueness;
- enforces positive quantities, optional nonnegative overrides, and pricing-mode consistency;
- adds tenant-oriented list and lookup indexes;
- reuses the existing `set_updated_at()` trigger function;
- preserves every previous catalog, customer, quote, reservation, identity, and audit record.

## Public Catalog

Each tenant owns one `public_catalog_settings` row. A catalog is publicly resolvable only when both the tenant and the catalog are active/enabled:

```text
GET /api/v1/public/catalogs/{tenantSlug}
        │
        ├── active tenant
        ├── enabled catalog
        ├── active + ready + published packages
        └── active + published resources when show_resources=true
```

Disabled and missing catalogs share the same public not-found boundary. This avoids exposing whether an organization slug exists but has chosen not to publish.

### Publication model

```text
tenants
  └── public_catalog_settings

packages
  ├── public_visible
  ├── public_featured
  └── public_sort_order

resources
  ├── public_slug
  ├── public_description
  ├── public_image_url
  ├── public_visible
  ├── public_featured
  └── public_sort_order
```

Resource public slugs are unique inside the tenant. Publishing requires an active resource and valid slug. Featuring requires visibility. Archiving a package/resource automatically removes public visibility so stale links return not found.

### Anonymous availability

```text
public package slugs + quantities + period
                    │
                    ▼
package quote-template expansion
                    │
                    ▼
existing reservation-aware availability engine
                    │
                    ▼
privacy-reduced public result
```

The public result contains resource names, requested quantities, and `can_fulfill`. It omits resource IDs, asset IDs, asset tags, serial numbers, eligible-asset counts, reserved counts, and available counts.

Availability is advisory. A quote request does not block inventory. The existing accepted-quote-to-reservation transaction remains the authoritative capacity boundary.

### Quote-request workflow

```text
Anonymous visitor
  │ package selection + period + contact + consent
  ▼
quote_requests
  ├── quote_request_packages  package and template snapshots
  └── quote_request_items     ordinary commercial line snapshots
  │
  ▼ authenticated tenant staff
NEW → IN_REVIEW → CLOSED / SPAM
  │
  └── convert
       ├── create or reuse same-tenant customer
       ├── create DRAFT quote
       ├── copy line snapshots
       └── mark request CONVERTED
```

Conversion locks the request and commits customer resolution, quote creation, quote items, and conversion state in one PostgreSQL transaction. Only `NEW` and `IN_REVIEW` requests may convert.

Every request snapshots:

```text
package names/slugs/quantities/templates
resource lines, quantities, prices, discounts
estimated totals and currency
availability result
terms text and terms version
consent state
```

Changing the storefront, package, price, or terms later does not rewrite historical requests.

### Anonymous-write controls

All anonymous `POST` routes still pass the global CSRF middleware. The quote-request service additionally applies:

```text
strict body and period validation
contact + terms-consent requirement
honeypot acknowledgement without persistence
five persisted requests / tenant / origin fingerprint / hour
salted SHA-256 fingerprint; raw IP is not stored
500-character User-Agent ceiling
```

`PUBLIC_REQUEST_FINGERPRINT_SALT` must be explicit and at least 32 characters outside local development.

### Authorization

```text
public_catalog.read     OWNER | ADMIN | MANAGER | STAFF
public_catalog.manage   OWNER | ADMIN
quote_request.read      OWNER | ADMIN | MANAGER | STAFF
quote_request.manage    OWNER | ADMIN | MANAGER
```

The public read/submit routes do not accept a tenant ID from the browser; they resolve an enabled tenant by slug. Administrative routes derive `tenant_id` exclusively from authenticated membership context.

## Migration 008

`008_public_catalog.sql`:

- creates `public_catalog_settings`;
- adds package/resource publication fields and consistency constraints;
- creates tenant-unique public resource slugs;
- creates `quote_requests`, `quote_request_packages`, and `quote_request_items`;
- adds composite tenant foreign keys for request snapshots and conversion references;
- enforces period, contact, consent, currency, amount, and conversion consistency;
- adds request status/rate-limit indexes and update triggers;
- preserves all v0.8 records.

## Quote Portal

Each tenant owns one `quote_portal_settings` row. The settings are read through `quote.read` and changed through `quote.manage`. Sending a quote requires an enabled portal and snapshots the current customer-facing copy, color, policy, terms text, and terms version into a single portal revision.

### Token lifecycle and transport

```text
32 cryptographically random bytes
          │ Base64URL without padding
          ▼
raw token ──────────────── returned once as /q#<token>
          │
          ├── browser sessionStorage for the current tab
          ├── history.replaceState removes the fragment
          └── X-RentStage-Quote-Token request header

SHA-256(raw token)
          └── persisted as quote_portals.token_hash
```

The raw value is absent from PostgreSQL, URL paths, query strings, API logs, audit metadata, and quote-detail reads. Rotation replaces the hash and increments `revision`, making the previous link return the same public not-found response as a fabricated token. Revocation closes the current revision without changing the sent quote, allowing a later reissue.

The `/q` document and public API responses use `no-store`, `no-referrer`, `noindex`, frame denial, and content-type protections. External tenant logos use `referrerPolicy=no-referrer`.

### Public route boundary

```text
GET  /api/v1/public/quote-portal
POST /api/v1/public/quote-portal/accept
POST /api/v1/public/quote-portal/reject
```

All three require the bearer header. The two unsafe operations also require the normal CSRF cookie/header pair. They never create an authenticated identity or accept `tenant_id` from the browser.

A disabled global portal setting makes every existing public link unavailable as well as preventing new issuance. `ACTIVE`, `ACCEPTED`, `REJECTED`, `REVOKED`, and `EXPIRED` are the only portal states.

### Customer decision evidence

Each issued portal freezes:

```text
headline and introduction
accent color
rejection policy
response-name policy
terms text and terms version
expiration timestamp
```

A customer decision records:

```text
decision timestamp and source
response name and optional response email
optional rejection reason
salted SHA-256 origin fingerprint
Unicode-safe 500-character User-Agent snapshot
portal event + application audit event
```

Raw IP addresses are not persisted. Repeated acceptance or rejection after the same completed decision returns the original result idempotently and does not create another reservation or duplicate high-level audit event.

### Transactional online acceptance

```text
BEGIN
  resolve token hash and enabled tenant portal
  lock quote row
  lock quote portal row
  require portal ACTIVE and quote SENT
  lock requested resources deterministically
  recalculate temporal availability
  insert PENDING reservation and historical line snapshots
  insert initial reservation status history
  update quote to ACCEPTED
  update portal to ACCEPTED with customer evidence
  insert portal event
COMMIT
```

The quote row is locked before the portal row so public reads/decisions use the same order as administrator quote-state changes and the synchronization trigger. Resource locks remain deterministic through the existing availability core.

An availability conflict commits only a privacy-reduced `ACCEPTANCE_BLOCKED` event. The quote remains `SENT`, the portal remains `ACTIVE`, and the response contains only resource name, requested quantity, and `can_fulfill`; internal IDs and raw capacity values remain private.

Administrator acceptance/rejection/cancellation/expiration updates the portal through `sync_quote_portal_from_quote_status()`. A manual administrative acceptance intentionally does not auto-create a reservation; the existing protected conversion action remains available. Customer acceptance is the atomic auto-conversion path.

### Administrative routes

```text
GET    /api/v1/quote-portal-settings                 quote.read
PATCH  /api/v1/quote-portal-settings                 quote.manage
POST   /api/v1/quotes/{quoteID}/send                 quote.manage
POST   /api/v1/quotes/{quoteID}/portal/reissue       quote.manage
DELETE /api/v1/quotes/{quoteID}/portal               quote.manage
```

The raw public URL appears only in successful send/reissue responses. Those mutation responses are explicitly non-cacheable. Later quote reads expose status, revision, expiry, views, terms version, and response evidence—but not the bearer link.

## Migration 009

`009_quote_portal.sql`:

- creates `quote_portal_settings`, `quote_portals`, and `quote_portal_events`;
- stores only unique SHA-256 token hashes;
- snapshots customer-facing presentation and terms per portal revision;
- constrains statuses, decision sources, evidence lengths, validity, and colors;
- adds tenant/status, expiration, portal-event, and quote-event indexes;
- reuses `set_updated_at()` triggers;
- creates the administrator quote-status synchronization trigger;
- seeds default settings for every existing tenant without changing quotes;
- preserves all v0.9 business and identity records.

## Current v0.10 boundaries

- Quote delivery is still manual copy/share. Transactional email, delivery receipts, reminders, and message templates are deferred.
- A public bearer link authorizes only one quote portal. It is not an account, session, workspace membership, or reusable customer identity.
- Online acceptance creates a `PENDING` reservation but does not collect payment, signature images, government identity, or legally qualified electronic signatures.
- Public availability is re-evaluated at acceptance time, but external services, staff capacity, delivery routes, and venue constraints remain outside the inventory engine.
- Application evidence is operational traceability; production legal/compliance requirements for consent, retention, jurisdiction, and electronic signatures require business/legal review.
- Public images remain external URLs; managed upload, transformation, malware scanning, and CDN delivery are deferred.
- Platform/WAF rate limiting, abuse monitoring, transactional communications, payments, WhatsApp, and AI orchestration remain deployment/product follow-ons.


## Billing & Payments Core

### Domain separation

```text
Quote         commercial proposal
Reservation   inventory/operations commitment
Invoice       internal receivable and tax snapshot
Payment       cash movement allocated to invoices
Deposit       refundable/retainable reservation guarantee
DTE           future external fiscal document/provider state
```

No billing record changes the historical quote or reservation snapshots that produced it. An invoice stores its own seller, customer, prices, tax categories, rates, and calculated amounts.

### Invoice lifecycle

```text
DRAFT
  ├── edit lines/customer/dates/tax mode
  ├── VOID
  └── issue under tenant settings lock
         ↓
       ISSUED
         ├── PARTIALLY_PAID
         │       └── PAID
         └── VOID only while paid_amount = 0
```

`OVERDUE` is a derived display state for open invoices whose `due_date` is before the current date. It is not stored as an additional database lifecycle state. Drafts have no final invoice number. Issuing locks the tenant billing-settings row, consumes `next_invoice_number`, writes the seller snapshot, derives fiscal readiness, appends an invoice event, and commits atomically.

### Tax calculation boundary

Each invoice item snapshots:

```text
quantity
unit price
discount
gross amount
net amount
tax code/category/rate
tax amount
line total
```

Supported categories are `TAXABLE`, `EXEMPT`, and `NON_TAXABLE`. Monetary business rules round to cents through integer-cent calculations; quantity and unit-price precision remain available at the input boundary. The tenant can configure prices as tax-exclusive or tax-included.

Initial tenant rules:

```text
IVA          TAXABLE      13.00%
EXEMPT       EXEMPT        0.00%
NON_TAXABLE  NON_TAXABLE   0.00%
```

These records are configuration data, not a claim that every rental line is legally taxable. The tenant/administrator remains responsible for assigning the appropriate rule.

### Source-to-invoice workflow

```text
accepted quote ─────┐
                    ├── load historical commercial lines
active reservation ┘   allocate header discount
                        add extra charges
                        apply selected/default tax rule
                        create independent DRAFT invoice

manual invoice ─────── manually supplied lines and tax rules
```

A partial unique index permits at most one active invoice per source quote and per source reservation. Voiding the invoice releases that source for a replacement invoice while preserving the void record.

### Payment transaction

```text
BEGIN
  normalize payment and require exact allocation sum
  sort invoice UUIDs
  lock each open invoice deterministically
  require same tenant, customer, currency
  require allocation <= current balance
  insert payment and allocations
  increment paid_amount
  derive ISSUED / PARTIALLY_PAID / PAID
  append invoice events
COMMIT
```

Voiding a confirmed payment locks the payment and allocated invoices, reverses every paid balance and state, appends reversal events, and marks the payment `VOIDED` in one transaction. Allocation rows remain as immutable historical links.

### Security deposits

Deposits are linked to a non-cancelled reservation and its customer but remain outside invoice totals, payments, output-tax estimates, and revenue reports.

```text
PENDING → RECEIVED → PARTIALLY_SETTLED
                     ├── RETURNED
                     ├── RETAINED
                     └── SETTLED (mixed return/retention)
```

Settlement inputs represent cumulative returned and retained totals. The database-generated balance is `amount - returned_amount - retained_amount`.

### Financial dashboard

The tenant-scoped read model derives:

```text
issued total
collected total
outstanding balance
overdue balance
estimated output tax
deposits currently held
draft/open/overdue/paid counts
recent invoices and payments
six-month billing and collection series
```

All records are constrained to the tenant base currency in v0.11. Multi-currency accounting is intentionally deferred.

### Authorization

```text
OWNER    billing/payment read + manage
ADMIN    billing/payment read + manage
MANAGER  billing/payment read + manage
STAFF    billing/payment read only
```

The UI filters navigation and mutation controls, but the authoritative boundary remains the Go permission middleware followed by tenant-scoped repository queries and composite foreign keys.

## Migration 010

`010_billing_payments.sql`:

- adds customer tax identity, registration, and billing-address fields;
- creates tenant billing settings and tax rules;
- creates invoice headers, immutable line snapshots, and invoice events;
- creates payments and invoice allocations;
- creates reservation security deposits;
- adds monetary, lifecycle, source-consistency, and tenant-link constraints;
- seeds settings and three initial tax categories for existing tenants;
- preserves all v0.10 records.

## Current v0.11 boundaries

- Internal invoices are not Hacienda DTEs and do not have a reception seal.
- `READY_FOR_DTE` records profile readiness only; it does not mean submitted or accepted by Hacienda.
- Purchase invoices, supplier expenses, input-tax credits, purchase books, and F-07 preparation are not modeled.
- Tax-rule administration is read-only in the v0.11 UI.
- Invoice printing is browser-rendered HTML, not a certified fiscal representation.
- Payment gateways, bank reconciliation, refunds, customer credit balances, and multi-currency ledgers are deferred.
- DTE signing/transmission, contingency, invalidation, credit/debit notes, and Secret Manager references belong to v0.12.

## El Salvador DTE Integration Foundation

### Domain separation

```text
Quote         proposal
Reservation   operational commitment
Invoice       internal receivable + immutable fiscal snapshots
DTE           fiscal numbering, payload, provider state, seal, and evidence
Payment       cash allocation
Deposit       reservation guarantee
```

DTE preparation requires an issued invoice. It reads only invoice snapshots, so later edits to tenant or customer profiles cannot rewrite fiscal history.

### Provider modes

```text
MOCK
  environment: TEST only
  network: none
  signature/seal: deterministic simulation
  fiscal validity: none

MH_HTTP
  endpoints: tenant configuration from official onboarding
  credentials: env:// references resolved at runtime
  flow: authenticate → sign → receive → persist redacted evidence
```

The provider interface keeps local domain transitions independent from one external implementation. Actual credentials are not stored in `dte_settings` or `dte_documents`.

### Preparation transaction

```text
BEGIN
  lock issued invoice
  require eligible invoice/fiscal state
  lock tenant DTE settings and next sequence
  validate provider/environment/document type
  build payload from immutable invoice snapshots
  allocate generation code + control number + idempotency key
  insert dte_document and PREPARED event
  consume sequence
COMMIT
```

Cancelled, rejected, and invalidated control numbers remain historical and are never reused.

### Submission and retry

```text
BEGIN
  lock READY_TO_SIGN / RETRY_REQUIRED DTE
  enforce attempt budget
  mark SUBMITTING
  synchronize invoice fiscal state
  append event
COMMIT

provider call outside transaction

BEGIN
  lock the same SUBMITTING DTE
  persist redacted request/response, signed document, seal, errors
  derive ACCEPTED / REJECTED / RETRY_REQUIRED
  synchronize invoice fiscal state
  append event
COMMIT
```

A retry uses the same identity and immutable payload. `next_attempt_at` is advisory until a background scheduler exists.

### Invalidation

Only `ACCEPTED` documents may start invalidation. The original issue request/response and receipt seal are never overwritten; invalidation request/response evidence is stored separately.

```text
ACCEPTED → INVALIDATION_PENDING
             ├── accepted → INVALIDATED + invoice fiscal VOIDED
             └── failed   → ACCEPTED with error evidence
```

### Outbound security

The configurable HTTP provider enforces:

```text
HTTP/HTTPS schemes only
HTTPS mandatory in PRODUCTION
no URL userinfo credentials
no localhost or non-public destination IPs
DNS and redirect target revalidation
maximum three redirects
TLS 1.2 minimum
30-second timeout
4 MiB response limit
no environment proxy inheritance
recursive credential redaction in persisted evidence
```

Infrastructure egress allowlisting remains required for production defense in depth.

### Authorization

```text
OWNER    fiscal.read + fiscal.manage
ADMIN    fiscal.read + fiscal.manage
MANAGER  fiscal.read + fiscal.manage
STAFF    fiscal.read
```

UI controls are convenience only. The Go permission middleware and tenant-filtered repositories are authoritative.

## Migration 011

`011_dte_integration.sql`:

- adds DTE receiver identity/activity/geography fields to customers;
- adds issuer geography settings and immutable issuer/receiver snapshots to invoices;
- adds item type, unit code, and product code snapshots to invoice lines;
- creates `dte_settings`, `dte_documents`, and `dte_events`;
- enforces supported provider/environment/document/status values;
- makes generation code, control number, and idempotency key unique per tenant;
- permits at most one active DTE per invoice;
- seeds `MOCK / TEST` settings for every existing tenant;
- preserves all v0.11 records.

## Current v0.12 boundaries

- MOCK signatures and seals are simulations with no fiscal validity.
- MH_HTTP has not been independently certified against a live taxpayer's current DGII onboarding contract.
- Submission, retry, and invalidation are explicit administrative operations; no background worker or stuck-state recovery scheduler exists.
- `query_url` is reserved but automatic provider reconciliation is not implemented.
- No certified legible PDF, QR, or public-query link is generated.
- Contingency, credit/debit notes, purchase documents, input-tax credit, books, and F-07 preparation are outside v0.12.
- Secret Manager integration is through secure environment-variable injection, not a direct SDK resolver.

## Production transition

The emulator is replaced, not redesigned:

```text
Local                           GCP staging/production
────────────────────────────    ─────────────────────────────
Auth emulator                   Firebase Auth / Identity Platform
FIREBASE_AUTH_EMULATOR_HOST     removed
Demo project ID                 real GCP/Firebase project ID
COOKIE_SECURE=false             COOKIE_SECURE=true
REQUIRE_VERIFIED_EMAIL=false    REQUIRE_VERIFIED_EMAIL=true
Local bootstrap enabled         disabled
Local ports                     HTTPS domains / Cloud Run
Local password in .env          no bootstrap password
```

On Cloud Run, the Admin SDK should use Application Default Credentials from the service account rather than a downloaded service-account key.
