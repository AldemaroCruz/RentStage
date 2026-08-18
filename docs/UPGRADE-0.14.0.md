# Upgrade RentStage v0.13.6 → v0.14.0

v0.14.0 turns the existing product modules into a concise commercial demonstration for prospective users and CONAMYPE. It introduces no schema migration or cloud-infrastructure change.

## What changes

- `/demo` provides a guided seven-minute inventory → quote → reservation → invoice → payment journey.
- The demo seed adds an accepted package quote, confirmed reservation, issued invoice, and partial payment for the existing `AudioPro Demo` tenant.
- Dashboard and sidebar navigation expose the new journey.
- DTE remains explicitly `MOCK / TEST`; WhatsApp + AI is labeled as roadmap.
- Release metadata moves to `0.14.0`.

## Data behavior

The commercial scenario is inserted only where the existing runtime already enables `SEED_DEMO_DATA=true` (local Docker Compose and the configured staging demo). Stable UUIDs and conflict handling make repeated application starts idempotent.

There is no destructive migration. Production environments with demo seeding disabled receive no demo records. Application rollback does not automatically delete records already inserted into a demo database.

## Windows upgrade

From the repository root in PowerShell:

```powershell
git pull --ff-only

bash scripts/ci/check-version.sh
python scripts/ci/check-workflow-yaml.py
python scripts/ci/check-migrations.py
bash scripts/ci/check-sensitive-files.sh

pwsh `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\check-powershell-syntax.ps1

docker compose config | Out-Null
docker compose up --build --detach

pwsh `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\test-smoke-common.ps1
```

The GitHub `RentStage CI/CD` workflow remains the authoritative production-build, integration, security, staging-deployment, and smoke-test gate.

## Staging verification

After merging to `main` and the green deployment:

1. Sign in to the public staging web service.
2. Open **Demo guiada** from the sidebar.
3. Confirm the readiness score is 100% for the owner demo account.
4. Follow each module link and confirm the accepted quote, confirmed reservation, invoice `INV-140001`, and $150 payment.
5. Open the public catalog in a separate tab.
6. Confirm the DTE step says `MOCK · TEST` and the WhatsApp + AI card says roadmap.

## Rollback

Revert the v0.14.0 application commit and redeploy. The route and navigation disappear without a database rollback. Seeded demo rows remain by design; for a disposable local environment, `docker compose down --volumes` followed by a rebuild recreates the baseline dataset.
