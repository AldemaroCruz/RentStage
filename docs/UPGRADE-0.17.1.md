# Upgrade RentStage v0.17.0 → v0.17.1

v0.17.1 adds a disabled-by-default, create-only production infrastructure apply path. Applying the source update does not mutate GCP. Infrastructure changes occur only after the separate bootstrap, temporary-access, protected-review, and manual workflow sequence.

## Source validation before commit

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

bash scripts/ci/check-version.sh
python scripts/ci/check-workflow-yaml.py
python scripts/ci/check-environment-isolation.py
python scripts/ci/test_production_apply_plan.py
bash scripts/ci/check-sensitive-files.sh

terraform -chdir=infra/staging fmt -check -recursive
terraform -chdir=infra/staging init -backend=false -input=false
terraform -chdir=infra/staging validate

terraform -chdir=infra/production fmt -check -recursive
terraform -chdir=infra/production init -backend=false -input=false
terraform -chdir=infra/production validate

git diff --check
git status --short
```

## Expected infrastructure plans

- Staging: `0 to add, 0 to change, 0 to destroy`. The minimal Firebase role is production-only in this release.
- Production before its first apply: approximately `50 to add, 0 to change, 0 to destroy` with the currently reviewed inputs.

Stop if staging changes or production proposes update, replacement, destroy, a staging reference, or a resource outside the dedicated production project.

## Required configuration after push

1. Rerun `scripts/bootstrap-gcp-production.sh` in Cloud Shell.
2. Add `GCP_INFRA_APPLY_WIF_PROVIDER` and `GCP_INFRA_APPLY_SERVICE_ACCOUNT` to the protected GitHub `production` Environment.
3. Set `PRODUCTION_INFRA_APPLY_ENABLED=false` at repository level.
4. Run the read-only production plan again.
5. Follow `docs/PRODUCTION-APPLY-0.17.1.md` only after cost and resource review.

## Rollback of the source update

Before any production apply, reverting the v0.17.1 commit removes the apply workflow and leaves existing staging and production resources unchanged. The bootstrap may create one service account, one WIF pool/provider, and one state-bucket IAM binding; retain them until the review is closed, then remove them manually only if this apply path is abandoned.

After a production apply, do not use a source revert as infrastructure rollback. Disable the gate, revoke temporary roles, retain the versioned state, and use a new read-only plan to choose a reviewed recovery action.
