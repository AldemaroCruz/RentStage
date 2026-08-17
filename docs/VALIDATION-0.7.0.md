# RentStage v0.7.0 validation record

## Scope

This record covers the frontend-only **Team & Access UI refinement** built on the working v0.6.2 identity foundation.

## Source checks completed

- `VERSION` and the web package version are both `0.7.0`.
- The team page no longer references the undefined `panel-heading` class.
- Membership and invitation sections use the explicit `team-section-header` layout.
- Desktop table headers and member rows share one grid definition.
- Responsive layouts expose labels for role, status, and activity.
- Member count handles singular and plural forms.
- Membership and invitation statuses are mapped to Spanish display labels.
- Mutation and copy controls use explicit button types.
- No files under `apps/api` or `apps/api/internal/database/migrations` changed.

## Automated checks completed

```text
TypeScript/TSX syntax parse             PASS — 47 source files
Team page strict semantic harness       PASS
CSS parser                              PASS — no parse errors
Desktop render, 1728 px                 PASS — no horizontal overflow
Compact desktop/tablet render, 1100 px  PASS — labeled responsive card
Mobile render, 680 px                   PASS — stacked layout, no overflow
Changed-file boundary                   PASS
```

The visual harness renders the production CSS and representative team markup at all three viewports. In each case, `document.body.scrollWidth` equals the viewport width.

## Compatibility checks

- No database migration.
- No Go API source or route change.
- No payload, permission, session, CSRF, cookie, or tenant-isolation contract change.
- Existing PostgreSQL and Authentication emulator volumes remain compatible.

## Host-side Docker build

The packaging environment does not provide Docker Engine and could not install the web dependency graph for a complete `next build`. Run the authoritative production build on the RentStage host:

```powershell
docker compose build --no-cache web
docker compose up -d
docker compose ps -a
```

Then verify:

```powershell
(Invoke-WebRequest `
  http://127.0.0.1:3000/login `
  -UseBasicParsing).StatusCode
```

Expected result:

```text
200
```

## Manual visual acceptance

Desktop:

- section headings have consistent 22 px horizontal inset;
- title, explanatory text, table headers, values, and buttons align cleanly;
- member identity and email are readable without oversized whitespace;
- action headers and buttons align to the right;
- invitation rows follow the same typography and spacing system.

Responsive:

- below 1180 px, member rows become labeled cards;
- below 700 px, invite fields, member controls, actions, and invitation rows stack without horizontal overflow;
- no content touches panel borders or overlaps adjacent labels.
