# Upgrade RentStage v0.16.0 → v0.17.0

v0.17.0 introduces an isolated, review-only production infrastructure foundation. It changes Terraform addresses for staging but preserves the existing remote objects with `moved` declarations.

## Before committing

Run the repository checks and both Terraform validations. Do not run a staging apply merely to test this release.

## Staging migration gate

After the change reaches `main`:

1. Keep staging resumed if a remote plan needs to refresh Cloud SQL.
2. Run **Staging Infrastructure** with `operation=plan`.
3. Confirm Terraform reports resource-address moves into `module.platform`.
4. Require `0 to add`, no replacement marker, and `0 to destroy`.
5. Do not apply and stop the upgrade if any existing GCP resource would be recreated.

An apply containing only reviewed state moves is optional; the existing staging application pipeline does not require it to continue deploying.

## Production bootstrap

Create a new GCP project; do not reuse `banded-nimbus-505914-s3`.

From Google Cloud Shell:

```bash
cd ~/RentStage
git pull --ff-only

export PROJECT_ID="replace-with-production-project-id"
export GITHUB_REPOSITORY="AldemaroCruz/RentStage"
export REGION="us-east1"

bash scripts/bootstrap-gcp-production.sh
```

Create the `production` GitHub Environment with a required reviewer and only the emitted variables. Generate a unique 24–64 character alphanumeric `PRODUCTION_DATABASE_PASSWORD`; do not reuse staging credentials.

Run **Production Infrastructure Plan** manually from `main`. It cannot apply the result. Review the log in GitHub; the binary plan is intentionally not uploaded.

## Compatibility

This release adds no migration, API endpoint, application environment variable, runtime Meta request, real sender, customer message, session change, or tenant-data mutation. Existing localhost and staging deployments remain compatible.

## Rollback

Reverting the code restores the old staging root, but do not remove `moved.tf` after a state-move apply without first validating the resulting state addresses. No production resource rollback exists because v0.17.0 exposes no production apply.
