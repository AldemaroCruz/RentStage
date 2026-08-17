# RentStage v0.13.4 — CI gate alignment and Go security patch

## Purpose

This update resolves the four quality gates that remained red after the first
GitHub Actions run while preserving the already-green API, web, and Docker
integration suites.

## Changes

- Synchronizes `VERSION` and `apps/web/package.json` at `0.13.4`.
- Updates the operational Go toolchain from `1.26.5` to `1.26.6` in:
  - GitHub Actions;
  - the API Docker builder;
  - local module synchronization helpers;
  - the Makefile guidance;
  - the staging CI/CD runbook.
- Removes the provider-incompatible `deletion_policy = "ABANDON"` argument from
  `google_firebase_project.rentstage`.
- Adds a provider-independent Terraform safeguard:

  ```hcl
  lifecycle {
    prevent_destroy = true
  }
  ```

- Adds this release to `CHANGELOG.md`.
- Preserves old validation documents that mention Go 1.26.5 because those files
  describe the actual historical toolchain used for their releases.

## Applying the update

Extract the ZIP at the root of `rentstage-starter`, then run:

```powershell
powershell `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\upgrade-v0.13.4.ps1
```

The script is idempotent and refuses to modify unrecognized file shapes. It
calculates every change before writing and restores files already written if a
write operation fails.

After it reports success, remove the temporary installer files before the
commit:

```powershell
Remove-Item `
  .\scripts\upgrade-v0.13.4.ps1, `
  .\README-v0.13.4.txt `
  -Force
```

Keep `docs/HOTFIX-0.13.4.md` as the release record.

## Validation sequence

```powershell
powershell `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\sync-go-modules.ps1

docker compose build --no-cache api web

docker compose up -d

powershell `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\run-smoke-suite.ps1
```

When Terraform CLI is available locally:

```powershell
terraform -chdir=infra/staging fmt -check
terraform -chdir=infra/staging init -backend=false -input=false
terraform -chdir=infra/staging validate
```

The GitHub workflow performs the authoritative Terraform, Govulncheck, and
Trivy checks.

## Expected historical references

After patching, this command may still report Go 1.26.5 in old validation
records:

```powershell
git grep -n "1\.26\.5"
```

Those historical references are intentionally retained. Operational files must
all use Go 1.26.6.

## Compatibility

- No SQL migration.
- No API or UI behavior change.
- No `.env` change.
- No tenant data change.
- No GCP resources are created by applying this source update.
- Staging DTE remains `MOCK / TEST`.
