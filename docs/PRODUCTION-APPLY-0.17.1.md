# RentStage production infrastructure apply v0.17.1

This release introduces the first production **infrastructure** apply path. It does not deploy the API or web application, create a public URL, seed demo data, contact Meta, or place Meta credential values in Terraform.

## Safety boundary

The apply requires all of the following:

1. `main` at the reviewed commit.
2. The protected GitHub `production` Environment and its reviewer.
3. `PRODUCTION_INFRA_APPLY_ENABLED=true` for the reviewed run only.
4. The exact protected GCP project ID as a workflow input.
5. The literal confirmation `APPLY-PRODUCTION`.
6. A dedicated apply WIF pool restricted to `.github/workflows/infra-production-apply.yml@refs/heads/main`.
7. Temporary project roles granted immediately before the run.
8. A newly generated saved plan whose JSON contains only create, read, or no-op actions.

The binary plan is applied in the same ephemeral job and is never uploaded. Terraform plans can contain sensitive values, including the database password.

## Identities

- `rentstage-infra-prod`: read-only planning plus isolated state-bucket object access.
- `rentstage-infra-apply-prod`: isolated apply WIF identity, state access, and no persistent project mutation role.
- `rentstage-deploy-prod`: reserved for a later application-deployment release; no project role and no WIF binding.

Production API runtime receives a custom Firebase role with only:

- `firebaseauth.users.get`
- `firebaseauth.users.createSession`

## One-time v0.17.1 setup in Cloud Shell

Rerun the idempotent bootstrap after pulling v0.17.1:

```bash
cd ~/RentStage
git pull --ff-only

export PROJECT_ID="rentstage-prod-260819-fa06"
export GITHUB_REPOSITORY="AldemaroCruz/RentStage"
export REGION="us-east1"

bash scripts/bootstrap-gcp-production.sh
```

The output adds two protected GitHub Environment variables:

- `GCP_INFRA_APPLY_WIF_PROVIDER`
- `GCP_INFRA_APPLY_SERVICE_ACCOUNT`

Add them to the existing `production` Environment. Keep these repository variables false:

```text
PRODUCTION_INFRA_APPLY_ENABLED=false
PRODUCTION_DEPLOY_ENABLED=false
```

The existing production database secret and environment variables do not change.

## Required pre-apply review

Run **Production Infrastructure Plan** again from the v0.17.1 commit. A brand-new production state with the currently reviewed defaults should show approximately `50 to add, 0 to change, 0 to destroy`: v0.17.1 replaces the planned Firebase Auth Admin binding with one custom role and one binding. Stop if the plan contains any update, replacement, destroy, staging reference, unknown project, or unexpected service.

Review the current Google Cloud estimates for Cloud SQL, SSD storage, backups, Secret Manager, Artifact Registry, and network egress. A budget alert reports spend but does not cap it.

## Grant temporary apply access

In Cloud Shell, with the production project active:

```bash
cd ~/RentStage

export PROJECT_ID="rentstage-prod-260819-fa06"
export CONFIRM_PROJECT_ID="$PROJECT_ID"

gcloud config set project "$PROJECT_ID"
bash scripts/gcp/production-apply-access.sh grant
bash scripts/gcp/production-apply-access.sh status
```

The script grants only the reviewed control-plane roles needed to create the declared APIs, service accounts, custom role, IAM bindings, Artifact Registry, Cloud SQL, Firebase/Identity Platform, API key, and Secret Manager resources. These roles are temporary.

## Run the protected apply from PowerShell

```powershell
$repo = 'AldemaroCruz/RentStage'
$project = 'rentstage-prod-260819-fa06'

gh variable set PRODUCTION_INFRA_APPLY_ENABLED `
  --repo $repo `
  --body 'true'

gh workflow run infra-production-apply.yml `
  --repo $repo `
  --ref main `
  -f "confirm_project_id=$project" `
  -f 'confirmation=APPLY-PRODUCTION'

$runId = gh run list `
  --repo $repo `
  --workflow infra-production-apply.yml `
  --event workflow_dispatch `
  --limit 1 `
  --json databaseId `
  --jq '.[0].databaseId'

gh run watch $runId `
  --repo $repo `
  --exit-status
```

Approve only the expected protected-Environment request. Do not rerun an old failed job after changing code or variables; start a new workflow so it produces a new plan.

## Close access immediately

Disable the repository gate from PowerShell whether the workflow succeeds or fails:

```powershell
gh variable set PRODUCTION_INFRA_APPLY_ENABLED `
  --repo 'AldemaroCruz/RentStage' `
  --body 'false'
```

Revoke the temporary project roles from Cloud Shell:

```bash
cd ~/RentStage
export PROJECT_ID="rentstage-prod-260819-fa06"
export CONFIRM_PROJECT_ID="$PROJECT_ID"

bash scripts/gcp/production-apply-access.sh revoke
bash scripts/gcp/production-apply-access.sh status
```

The status output should contain no direct project role for `rentstage-infra-apply-prod`.

## Post-apply verification

1. Run **Production Infrastructure Plan** again and require `0 to add, 0 to change, 0 to destroy`.
2. Confirm Cloud SQL is `RUNNABLE`, deletion protection is enabled, connector enforcement is `REQUIRED`, and TLS mode is `ENCRYPTED_ONLY`.
3. Confirm the state object exists under `rentstage/production/` in the production-only bucket.
4. Confirm the six Meta secret containers have no versions.
5. Confirm `rentstage-deploy-prod` still has no direct project roles and cannot be impersonated from GitHub.
6. Confirm both production gates are false.

## Failure and recovery

- First disable the gate and revoke the temporary roles.
- Do not run `terraform destroy`, delete the state object, or manually delete Cloud SQL.
- Start a new read-only production plan and inventory the resources already recorded in state.
- If the provider stopped after creating only part of the graph, correct the cause and use a new reviewed create/no-op plan. The safety gate accepts partial recovery only when no update, replacement, or destroy is proposed.
- If a legitimate update becomes necessary, prepare a later reviewed increment that names that exact change. v0.17.1 intentionally rejects it.
- Use the versioned state bucket for state recovery; never copy state into the repository or a GitHub artifact.

There is no automatic rollback because deleting newly created infrastructure can destroy state or data. Cloud SQL deletion protection remains enabled.
