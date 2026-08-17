# Upgrade RentStage v0.6.2 → v0.7.0

## Scope

RentStage v0.7.0 is a frontend-only interface refinement for **Equipo y permisos**.

It does not change:

- PostgreSQL schema or data;
- Go API routes or payloads;
- Firebase Authentication configuration;
- HttpOnly sessions or CSRF behavior;
- roles, permissions, membership hierarchy, or tenant isolation.

## Files changed

```text
VERSION
README.md
CHANGELOG.md
apps/web/package.json
apps/web/app/settings/team/page.tsx
apps/web/app/globals.css
scripts/smoke-auth.ps1
docs/UPGRADE-0.7.0.md
docs/VALIDATION-0.7.0.md
```

## Apply on Windows PowerShell

Run from the existing `rentstage-starter` directory.

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

docker compose down

Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.7.0.zip" `
  -DestinationPath . `
  -Force

Get-Content .\VERSION
```

Expected version:

```text
0.7.0
```

Build the changed frontend image:

```powershell
docker compose build --no-cache web
docker compose up -d
docker compose ps -a
```

The API, database, and Authentication emulator do not require a source rebuild for this release. Rebuilding all services is safe but unnecessary.

## Verify

```powershell
(Invoke-WebRequest `
  http://127.0.0.1:3000/login `
  -UseBasicParsing).StatusCode
```

Expected:

```text
200
```

Sign in with the configured local owner account and open:

```text
http://127.0.0.1:3000/settings/team
```

Confirm:

1. `MEMBRESÍAS`, its title, and description are inset consistently inside the panel.
2. User, role, status, activity, and action labels align with their corresponding values.
3. One member renders as `1 miembro`, not `1 miembros`.
4. Invitation headings use the same padding and vertical rhythm as memberships.
5. At narrower browser widths, each membership becomes a labeled responsive card instead of a clipped horizontal row.
6. Login, logout, workspace switching, invitation creation, role changes, suspension, and reactivation retain their previous behavior.

## Data preservation

Do not run:

```powershell
docker compose down -v
```

The `-v` flag deletes persistent PostgreSQL and Authentication emulator volumes. It is not required for v0.7.0.

## Rollback

Reapply the v0.6.2 full package or restore these frontend files from source control, then rebuild only `web`:

```powershell
docker compose build --no-cache web
docker compose up -d web
```

No database rollback is required.
