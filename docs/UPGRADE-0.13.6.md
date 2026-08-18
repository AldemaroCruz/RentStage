# Upgrade RentStage v0.13.5 → v0.13.6

v0.13.6 is a frontend and documentation update for staging demonstration readiness. It does not introduce a database migration or change GCP infrastructure.

## Preserved

- PostgreSQL and Firebase emulator volumes;
- Cloud SQL and Identity Platform users;
- tenants, inventory, packages, catalogs, quotes, reservations, billing, DTE evidence, and audit history;
- GitHub Environment variables and secrets;
- the local Docker Compose owner account;
- the public-web/private-API Cloud Run boundary.

## Changed

- local demo credentials are conditional on the Authentication emulator build flag;
- staging login fields start empty and the local account card is omitted;
- the dashboard release strip reflects the deployed product instead of v0.5;
- release metadata and validation documentation are synchronized at 0.13.6.

## Apply from PowerShell

From the repository root:

```powershell
git pull --ff-only

Get-Content .\VERSION

git diff --check

pwsh `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\check-powershell-syntax.ps1
```

Expected version:

```text
0.13.6
```

## Local validation

```powershell
docker compose config | Out-Null
docker compose build web

docker compose run `
  --rm `
  --no-deps `
  --entrypoint npm `
  web `
  test
```

The local login page must continue to show and prefill the documented owner account.

## Staging acceptance

After the normal `main` pipeline succeeds:

1. Open the staging login page in a private browser window.
2. Confirm both fields start empty.
3. Confirm the local credential card and `RentStage123!` are not rendered.
4. Sign in with the staging account and rotated staging password.
5. Confirm the dashboard release strip displays `v0.13.6` and DTE MOCK / TEST wording.
6. Confirm the complete staging smoke suite succeeds.

No Terraform apply is required specifically for this release.
