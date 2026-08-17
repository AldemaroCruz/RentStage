# Upgrade RentStage v0.12.1 -> v0.13.0

v0.13.0 adds the staging delivery foundation. It does not introduce a database migration and does not modify existing business data.

## Preserved

- `.env` and local ports;
- PostgreSQL and Firebase emulator volumes;
- tenants, users, memberships, inventory, packages, catalogs, quotes, reservations, billing, DTE evidence, and audit history;
- localhost as the development environment.

## New or changed

- GitHub Actions workflows and Dependabot;
- Terraform staging module;
- GCP bootstrap/deploy/rollback scripts;
- parameterized local/staging smoke helpers;
- Cloud Run service-to-service authentication in the Next.js proxy;
- stricter non-local API configuration validation;
- hardened Docker build stages and repository security policy;
- CI documentation and validation scripts.

## 1. Back up local PostgreSQL

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter
New-Item -ItemType Directory -Force .\backups | Out-Null

docker compose exec db `
  pg_dump `
  -U rentstage `
  -d rentstage `
  -Fc `
  -f /tmp/rentstage-v0.12.1-before-v0.13.0.dump

docker compose cp `
  db:/tmp/rentstage-v0.12.1-before-v0.13.0.dump `
  .\backups\rentstage-v0.12.1-before-v0.13.0.dump

Get-Item .\backups\rentstage-v0.12.1-before-v0.13.0.dump
```

## 2. Stop without deleting volumes

```powershell
docker compose down
```

Do not run `docker compose down -v`.

## 3. Apply the incremental update

```powershell
Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.13.0.zip" `
  -DestinationPath . `
  -Force

Get-Content .\VERSION
```

Expected:

```text
0.13.0
```

## 4. Rebuild local development

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a

Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
(Invoke-WebRequest http://127.0.0.1:3000/api/healthz -UseBasicParsing).StatusCode
```

## 5. Run all local tests

```powershell
powershell `
  -ExecutionPolicy Bypass `
  -File .\scripts\test-smoke-common.ps1

powershell `
  -ExecutionPolicy Bypass `
  -File .\scripts\run-smoke-suite.ps1
```

The complete suite covers authentication, packages, public catalog, Quote Portal, Billing & Payments, and DTE MOCK/TEST.

## 6. Commit to GitHub and create staging

Follow `docs/STAGING-CICD.md`. Do not create or upload a Google service-account JSON key; the workflows use Workload Identity Federation.
