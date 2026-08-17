# RentStage v0.10.0 validation record

## Scope

This record covers the **Quote Portal** update built on the runtime-validated RentStage v0.9.0 source tree.

Validated areas:

```text
migration 009 and schema invariants
bearer-token generation, hashing, rotation, and transport boundaries
portal settings and tenant permissions
public quote read model
accept/reject validation and idempotency design
transactional acceptance-to-reservation orchestration
availability-conflict privacy reduction
administrator status synchronization
Next.js public-route guards and no-cache/security headers
admin/public UI TypeScript contracts
upgrade-only archive composition and reproducibility
```

## Source baseline supplied by the workstation

The supplied v0.9.0 log demonstrates that the real Docker build completed for both API and web, TypeScript finished, and Next.js generated all 23 v0.9 routes. It also demonstrates healthy `db`, `auth`, and `api` containers, HTTP 200 from `/login`, applied migration 008, all four public-catalog tables, and passing authentication, package, public-catalog submission, and read-only smoke tests.

Version 0.10 is therefore treated as an incremental update over a confirmed v0.9 runtime rather than an unverified reconstruction.

## Go validation completed in the build environment

The current API source was copied into an isolated Go 1.23 contract harness. Only external Firebase Admin and pgx implementations were replaced by local API-compatible stubs; RentStage packages and tests are the actual v0.10 source.

Commands:

```bash
GOTOOLCHAIN=local GOPROXY=off go test -run '^$' ./...
GOTOOLCHAIN=local GOPROXY=off go test ./...
GOTOOLCHAIN=local GOPROXY=off go vet ./...
```

Result:

```text
PASS
```

Coverage includes existing authentication, identity, availability, customer, operations, packages, public catalog, quote, reservation, scheduling, and warehouse tests plus Quote Portal tests for:

- 256-bit Base64URL token generation and SHA-256 round trip;
- malformed-token rejection;
- settings normalization and validation;
- explicit terms acceptance;
- email/name/rejection-reason normalization;
- token-scoped salted origin fingerprints;
- Unicode-safe User-Agent truncation.

## TypeScript validation completed in the build environment

The current `app`, `components`, `lib`, and `next.config.ts` trees were copied into the existing strict framework-contract harness.

Command:

```bash
tsc -p tsconfig.json --pretty false
```

Result:

```text
PASS
```

A separate TypeScript transpilation pass processed every non-declaration `.ts` and `.tsx` source file with no syntax errors.

Validated v0.10 areas include:

- `/q` public route and fragment-token bootstrap;
- tab-scoped token storage and address cleanup;
- dedicated token request header;
- acceptance/rejection forms and validation responses;
- decision, expiry, rejection, and availability-conflict views;
- quote-detail send/reissue/revoke controls;
- one-time link banner and clipboard behavior;
- portal settings administration and read-only role behavior;
- quote/portal TypeScript response contracts;
- RootFrame public-route handling;
- Next.js proxy token/header forwarding;
- security headers configured for `/q`.

## Migration and persistence checks

Static review verified:

- migration order is `001` through `009`;
- `009_quote_portal.sql` is additive relative to all existing v0.9 tables;
- `quote_portal_settings` is one row per tenant;
- `quote_portals` is one row per tenant/quote and one row per token hash;
- bearer storage is `CHAR(64) token_hash`; no raw-token column exists;
- portal and event foreign keys use composite tenant relationships;
- statuses, decision sources, revision, validity, color, evidence lengths, and view counts are constrained;
- `set_updated_at()` is reused for settings and portals;
- default settings are inserted for existing tenants with `ON CONFLICT DO NOTHING`;
- the quote-status trigger updates only active portals;
- migration 009 does not delete or recreate existing tables or volumes.

A live PostgreSQL parser/application was not available in this build environment. The definitive migration check remains the workstation startup and `schema_migrations` query documented in `UPGRADE-0.10.0.md`.

## Transaction and concurrency review

The online acceptance path uses one caller-owned PostgreSQL transaction:

```text
resolve token + enabled portal
lock quote row
lock portal row
require SENT + ACTIVE + unexpired
lock resources deterministically
recalculate availability
insert reservation + line snapshots + initial status history
update quote to ACCEPTED
update portal evidence to ACCEPTED/CUSTOMER
insert portal event
commit
```

The quote-before-portal lock order matches administrator quote updates followed by the synchronization trigger. Token rotation uses the same portal row with a new unique hash and incremented revision. Repeated completed decisions return the existing result before conversion and are therefore idempotent.

When availability fails, reservation creation performs no writes. The transaction records only `ACCEPTANCE_BLOCKED`, commits that evidence, and returns the privacy-reduced conflict while leaving quote/portal active.

## Security review

Static checks confirmed:

- raw tokens are never inserted into SQL, audit metadata, portal events, paths, or query parameters;
- generated links use `/q#<token>`;
- the browser removes the fragment with `history.replaceState` and forwards the token through `X-RentStage-Quote-Token`;
- the Next.js same-origin proxy explicitly forwards the dedicated token header;
- public POST decisions remain behind the existing CSRF middleware;
- missing/malformed/replaced tokens share a public not-found boundary;
- global portal disable blocks reads and decisions for existing links;
- send/reissue mutation responses containing the one-time link use `Cache-Control: no-store`;
- public API responses use no-store/no-referrer/noindex/nosniff protections and vary by token header;
- `/q` uses no-store, no-referrer, noindex, frame denial, permissions restrictions, and CSP frame/base/form controls;
- external logo requests use `referrerPolicy=no-referrer`;
- raw IP addresses are not persisted;
- public availability conflicts omit internal IDs, asset metadata, and raw capacity counts;
- ordinary protected quote reads never return a bearer URL.

## Formatting and static checks

Completed before packaging:

- `gofmt` over all modified/new Go files;
- `git diff --check`;
- JSON parsing;
- Docker Compose YAML parsing;
- CSS delimiter balance;
- internal TypeScript alias/import resolution;
- migration filename/order checks;
- scan for secret/private files and generated directories;
- scan for obsolete token-in-path/query route patterns;
- PowerShell smoke-script static review.

## Runtime validation required on the workstation

This environment does not provide Docker Engine, PostgreSQL 18, the Firebase Authentication emulator, or PowerShell. It could not execute the real integration sequence:

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a

powershell -ExecutionPolicy Bypass -File .\scripts\smoke-auth.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-packages.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-public-catalog.ps1 -SkipSubmission
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-quote-portal.ps1
```

The workstation must additionally confirm:

- `009_quote_portal.sql` appears in `schema_migrations`;
- all three Quote Portal tables exist;
- the real Next.js build includes `/q` and `/settings/quote-portal`;
- old tokens fail after rotation;
- online acceptance creates exactly one reservation;
- online rejection creates none;
- global disable closes existing links;
- availability conflict leaves the quote sent and portal active;
- public responses do not disclose internal inventory data.

## Packaging checks

The final packaging step verifies:

- the update ZIP contains only files changed from the distributed v0.9.0 tree;
- `.env`, `.git`, `.next`, `node_modules`, backups, volumes, private keys, and credentials are excluded;
- archive paths are relative to the `rentstage-starter` project root;
- ZIP CRC integrity passes;
- applying the update to a fresh v0.9.0 reconstruction produces the same source tree as v0.10.0;
- a SHA-256 checksum is generated and independently verified.

Exact archive file count and digest are recorded in the distributed checksum file and final delivery message.

## Boundaries intentionally left open

- Quote links are copied manually; transactional email and delivery history are deferred.
- Operational evidence is not a jurisdiction-specific qualified electronic signature.
- Payments, deposits, refunds, taxes, and invoicing are not part of acceptance.
- CAPTCHA/WAF controls, managed public-file delivery, customer accounts, WhatsApp, and AI orchestration remain follow-ons.
