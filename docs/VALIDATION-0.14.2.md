# RentStage v0.14.2 validation record

## Release contracts

- [ ] `VERSION`, `apps/web/package.json`, README, and changelog report `0.14.2`.
- [ ] Workflow YAML, sensitive-file policy, shell syntax, PowerShell syntax, and `git diff --check` pass.
- [ ] No migration, API contract, GCP resource, IAM permission, tenant-data, or application-runtime change is introduced.

## Static validation

```bash
bash scripts/ci/check-version.sh
python scripts/ci/check-workflow-yaml.py
bash scripts/ci/check-sensitive-files.sh
bash -n scripts/gcp/staging-cost-control.sh
shellcheck scripts/gcp/staging-cost-control.sh
bash scripts/gcp/test-staging-cost-control.sh
git diff --check
```

## Workflow acceptance

1. Keep repository variable `STAGING_DEPLOY_ENABLED=true` and run `pause`; confirm it fails before Google authentication with an actionable gate message.
2. Set `STAGING_DEPLOY_ENABLED=false` and run `pause` with `confirm=false`; confirm it fails without changing Cloud SQL.
3. Run `pause` with `confirm=true`; confirm activation policy becomes `NEVER` and the Cloud Run services remain present.
4. Run `status`; confirm the job summary reports the paused policy, retained Cloud Run services, and disabled deployment gate.
5. Run `resume` with `confirm=true`; confirm activation policy becomes `ALWAYS`.
6. Re-enable `STAGING_DEPLOY_ENABLED`, run **RentStage CI/CD**, and confirm deployment plus the staging smoke suite pass.

Record evidence:

```text
Pause run: ______________________________
Status run: _____________________________
Resume run: _____________________________
CI/CD run: ______________________________
Commit SHA: _____________________________
Validated by: ___________________________
Date: __________________________________
```
