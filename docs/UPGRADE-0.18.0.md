# Upgrade RentStage v0.17.1 → v0.18.0

v0.18.0 adds an isolated local WhatsApp Cloud API contract harness. It does not require a Meta account and does not apply or deploy production infrastructure.

## Source validation

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

bash scripts/ci/check-version.sh
python scripts/ci/check-workflow-yaml.py
python scripts/ci/check-migrations.py
python scripts/ci/check-environment-isolation.py
python scripts/ci/test_production_apply_plan.py
bash scripts/ci/check-sensitive-files.sh

pwsh `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\check-powershell-syntax.ps1

Push-Location .\apps\web
try {
    npm install --no-audit --no-fund --package-lock=false
    npm run typecheck
    npm run test:coverage
}
finally {
    Pop-Location
}

git diff --check
git status --short
```

The existing API and Docker CI jobs also run `gofmt`, `go mod verify`, race tests, vet, migrations, and the complete smoke suite.

## Local runtime validation

```powershell
docker compose up --build --detach
bash scripts/ci/wait-local.sh

pwsh `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\smoke-meta-local.ps1
```

Expected result: raw challenge accepted, signed inbound processed once, identical redelivery reported as a duplicate, and the local Graph contract returns one `wamid.local.*` message ID.

## Data compatibility

Migration `014_meta_whatsapp_local_adapter.sql` adds a tenant-scoped connection table. Existing conversations and messages remain unchanged. With local demo seeding enabled, a deterministic local connection is inserted or updated. No credential value is persisted.

## Rollback

Disable `META_WHATSAPP_MODE` or revert the v0.18.0 source update. The additive table can remain safely unused; do not delete it while any WhatsApp conversation references are needed. Production and staging are unaffected because neither enables the adapter.
