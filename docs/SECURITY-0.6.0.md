# RentStage v0.6 security model

## Release boundary

Version 0.6 replaces the previous trusted local tenant/actor headers with authenticated server context and role-based authorization.

It is intended for:

```text
local multi-user testing
workspace-isolation validation
role and invitation testing
preparation for GCP staging
```

It is **not** a production deployment by itself because Docker Compose includes a Firebase Authentication emulator and development bootstrap credentials.

## Security goals

1. A request without a valid server session cannot access RentStage business data.
2. A valid user can only access active organizations where that user has an active membership.
3. A tenant identifier supplied or modified by the browser cannot grant access by itself.
4. Every protected operation must satisfy an explicit role permission.
5. Unsafe browser requests must satisfy CSRF validation.
6. Membership-management hierarchy must be stricter than the generic team permission.
7. Invitation tokens must not be stored in recoverable form.
8. Audit records should identify the real authenticated user.
9. Suspension should take effect without waiting for the Firebase session cookie to expire.
10. Existing cross-tenant database relationships must remain impossible.

## Trust boundaries

```text
Untrusted
────────────────────────────
Browser inputs
URL parameters
request body tenant IDs
cookies before membership validation
invitation raw token
Firebase ID token before verification

Trusted after verification
────────────────────────────
Firebase Admin decoded identity
RentStage user status
active membership from PostgreSQL
server-selected tenant context
role-to-permission result
repository tenant filters
PostgreSQL constraints
```

## Authentication controls

### ID-token exchange

The browser authenticates with Firebase and sends the resulting ID token to RentStage only through the session endpoint.

The API:

```text
verifies signature/emulator mode
checks token revocation state
requires auth_time within five minutes
loads current Firebase user state
rejects disabled accounts
optionally requires verified email
synchronizes the user mirror
creates a bounded server session cookie
```

The browser then clears Firebase client authentication state and relies on the HttpOnly server cookie.

### Session verification

Every authenticated API request uses revocation-aware session-cookie verification. RentStage additionally checks its own `users.status` on each request.

### Cookie policy

Local:

```text
HttpOnly=true
SameSite=Lax
Secure=false
```

Cloud:

```text
HttpOnly=true
SameSite=Lax or stricter after domain review
Secure=true
HTTPS only
```

No JavaScript code can read the session or tenant cookie.

## CSRF controls

RentStage uses a random double-submit token:

```text
GET /api/v1/auth/csrf
  ├── sets HttpOnly SameSite=Strict cookie
  └── returns the same random token in JSON

POST/PATCH/PUT/DELETE /api/*
  ├── browser sends cookie automatically
  └── frontend sends token in X-CSRF-Token
```

The API compares both values in constant time and returns HTTP 403 on mismatch.

CSRF is applied to session creation and logout as well as business mutations.

## Tenant context controls

The selected tenant cookie is intentionally not cryptographically signed as an authorization claim because it is never trusted alone.

For each tenant-scoped request:

```text
1. Verify authenticated session.
2. List ACTIVE memberships for the user.
3. Read selected tenant preference.
4. Match it against the active membership list.
5. Fall back only to another authorized workspace.
6. Put tenant and role into server context.
7. Execute permission middleware.
8. Pass context tenant to repositories.
```

Changing the cookie to another organization's UUID cannot create a membership.

## Authorization controls

### Permission middleware

Every business route declares one required permission at registration time.

Examples:

```text
POST /resources                         catalog.manage
PATCH /assets/{id}                      inventory.manage
POST /quotes                            quote.manage
POST /reservations                      reservation.manage
POST /reservations/{id}/checkout        warehouse.operate
GET /audit                              audit.read
PATCH /tenant                           tenant.manage
POST /team/invitations                  team.manage
```

### Membership hierarchy

Even an actor with `team.manage` is subject to hierarchy:

```text
OWNER  may manage ADMIN, MANAGER, STAFF
ADMIN  may manage MANAGER and STAFF only
```

An ADMIN cannot invite another ADMIN, change an OWNER, or change another ADMIN.

The final active owner cannot be demoted or suspended through the repository command.

### UI controls are not authoritative

The web application filters navigation, guards routes, and hides mutation buttons, but the API middleware remains the security boundary.

## Invitation security

Invitation token generation:

```text
32 cryptographically random bytes
Base64URL encoding for the acceptance URL
SHA-256 hash persisted in PostgreSQL
raw token returned only when created
seven-day expiry
```

Additional controls:

- one pending invitation per tenant/email;
- existing memberships cannot be invited again;
- expired pending invitations are persisted as expired before a replacement is created;
- acceptance requires authenticated email equality;
- acceptance locks the invitation row;
- accepted and revoked invitations cannot be reused;
- invitation role cannot be OWNER;
- acceptance writes membership, invitation state, and active preference transactionally.

The current local release displays a copyable URL instead of sending email. Email delivery must not log or persist the raw token in future versions.

## Suspension semantics

Membership suspension is enforced at tenant resolution, not merely at login.

```text
Firebase session remains valid
          │
          ▼
Membership becomes SUSPENDED
          │
          ▼
Next tenant-scoped request cannot establish context
```

This allows immediate organization-level revocation without disabling the person's entire RentStage identity.

## Audit controls

Mutation history records the authenticated RentStage user UUID. Audit reads join the current user profile to display name and email.

Audit data remains tenant-scoped and requires `audit.read`.

Sensitive material that must never be included in audit metadata:

```text
passwords
Firebase ID tokens
session cookies
CSRF tokens
invitation raw tokens
service-account credentials
```

## Database isolation controls

Existing domain tables retain tenant ownership and composite tenant foreign keys.

Examples:

```text
resource category must belong to the same tenant
asset resource must belong to the same tenant
quote customer and resources must belong to the same tenant
reservation customer/resources/assets must belong to the same tenant
```

Identity tables add:

```text
unique identity UID
unique normalized email
membership primary key (tenant_id, user_id)
one pending invitation per tenant + normalized email
foreign keys for inviter/acceptor/preferences
```

## Local emulator warning

The Authentication emulator issues development tokens and has no production security boundary. The emulator ports are bound to `127.0.0.1` in Compose, but they must still never be forwarded publicly.

Never deploy with:

```text
FIREBASE_AUTH_EMULATOR_HOST set
LOCAL_AUTH_BOOTSTRAP=true
RentStage123! as a real password
COOKIE_SECURE=false over the internet
```

## GCP staging requirements

Before remote pilot access:

### Identity

```text
real Firebase Authentication / Identity Platform project
email/password provider configured
verified email required
local bootstrap disabled
emulator environment variable removed
```

### Runtime

```text
Cloud Run HTTPS endpoints
COOKIE_SECURE=true
strict production CORS allowlist
separate web and API service accounts
minimum IAM roles
Secret Manager for sensitive configuration
Application Default Credentials
```

### Database

```text
Cloud SQL PostgreSQL
private or controlled connector path
automated backups + PITR
SSL/connector enforcement
separate migration identity if practical
least-privilege database user
```

### Edge and abuse controls

```text
rate limits for login/session/invitation endpoints
Cloud Armor or equivalent when justified
request-body limits
security headers
bot/credential-stuffing monitoring
invitation creation throttling
structured security logs
alerting on repeated 401/403/CSRF failures
```

### Operational controls

```text
budget alerts
Cloud Logging retention
Cloud Monitoring uptime checks
session duration review
incident response procedure
backup restore test
cross-tenant automated regression suite
```

## Deferred security work

The following are intentionally outside v0.6 and should be tracked before general availability:

- production email verification and password reset UX;
- MFA for privileged roles;
- Google/OIDC login;
- user-wide session revocation UI;
- organization ownership transfer workflow;
- removal/reactivation lifecycle beyond suspension;
- API rate limiting;
- Content Security Policy and complete security-header policy;
- production invitation email delivery;
- terms/privacy/consent records;
- account deletion/export workflows;
- centralized security analytics;
- formal penetration testing.

## Recommended authorization regression matrix

Automate at least these cases against PostgreSQL and a running identity test environment:

```text
unauthenticated request                              → 401
invalid/revoked session                              → 401
active user without any workspace                    → workspace_required / onboarding
active user requesting unowned tenant cookie         → authorized fallback or 403
suspended membership                                 → no tenant access
STAFF modifies quote                                 → 403
STAFF performs warehouse return                      → allowed
MANAGER changes resource price                       → 403
MANAGER creates reservation                          → allowed
ADMIN modifies OWNER                                 → 403
ADMIN invites ADMIN                                  → 422 validation
OWNER suspends own membership                        → 422 validation
attempt to remove final active owner                 → 409
Tenant B reads Tenant A customer UUID                → 404
invitation accepted by different email               → 403
expired/revoked/accepted invitation reused           → 404
unsafe request without CSRF                          → 403
```
