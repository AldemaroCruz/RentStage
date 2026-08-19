#!/usr/bin/env bash
set -euo pipefail

: "${PROJECT_ID:?Set PROJECT_ID to the new billing-enabled production project ID}"
: "${GITHUB_REPOSITORY:?Set GITHUB_REPOSITORY to owner/repository}"

REGION="${REGION:-us-east1}"
POOL_ID="${POOL_ID:-github-actions-production}"
PROVIDER_ID="${PROVIDER_ID:-rentstage-production}"
APPLY_POOL_ID="${APPLY_POOL_ID:-github-actions-production-apply}"
APPLY_PROVIDER_ID="${APPLY_PROVIDER_ID:-rentstage-production-apply}"
INFRA_SA_ID="${INFRA_SA_ID:-rentstage-infra-prod}"
APPLY_SA_ID="${APPLY_SA_ID:-rentstage-infra-apply-prod}"
DEPLOY_SA_ID="${DEPLOY_SA_ID:-rentstage-deploy-prod}"
STATE_BUCKET="${STATE_BUCKET:-${PROJECT_ID}-rentstage-tfstate}"
APPLY_WORKFLOW_REF="${GITHUB_REPOSITORY}/.github/workflows/infra-production-apply.yml@refs/heads/main"

command -v gcloud >/dev/null || { echo "gcloud is required" >&2; exit 1; }
command -v gh >/dev/null || { echo "GitHub CLI is required and must be authenticated" >&2; exit 1; }

REPO_ID="$(gh api "repos/${GITHUB_REPOSITORY}" --jq '.id')"
OWNER_ID="$(gh api "repos/${GITHUB_REPOSITORY}" --jq '.owner.id')"
[[ "$REPO_ID" =~ ^[0-9]+$ && "$OWNER_ID" =~ ^[0-9]+$ ]] || {
  echo "Unable to resolve immutable GitHub repository and owner IDs" >&2
  exit 1
}

gcloud config set project "$PROJECT_ID" >/dev/null

BILLING_ENABLED="$(
  gcloud billing projects describe "$PROJECT_ID" \
    --format='value(billingEnabled)'
)"
if [[ "$BILLING_ENABLED" != "True" && "$BILLING_ENABLED" != "true" ]]; then
  echo "Project ${PROJECT_ID} does not have billing enabled." >&2
  exit 1
fi

gcloud services enable \
  serviceusage.googleapis.com \
  cloudresourcemanager.googleapis.com \
  iam.googleapis.com \
  iamcredentials.googleapis.com \
  sts.googleapis.com \
  storage.googleapis.com \
  --quiet

PROJECT_NUMBER="$(
  gcloud projects describe "$PROJECT_ID" \
    --format='value(projectNumber)'
)"

ensure_service_account() {
  local account_id="$1"
  local display_name="$2"
  local email="${account_id}@${PROJECT_ID}.iam.gserviceaccount.com"

  gcloud iam service-accounts describe "$email" >/dev/null 2>&1 ||
    gcloud iam service-accounts create "$account_id" \
      --display-name "$display_name"
}

ensure_service_account "$INFRA_SA_ID" "RentStage production infrastructure plan"
ensure_service_account "$APPLY_SA_ID" "RentStage production infrastructure apply"
ensure_service_account "$DEPLOY_SA_ID" "RentStage production deployment (reserved)"

INFRA_SA="${INFRA_SA_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
APPLY_SA="${APPLY_SA_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
DEPLOY_SA="${DEPLOY_SA_ID}@${PROJECT_ID}.iam.gserviceaccount.com"

for role in \
  roles/viewer \
  roles/serviceusage.serviceUsageConsumer; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:${INFRA_SA}" \
    --role="$role" \
    --condition=None \
    --quiet >/dev/null
done

if ! gcloud storage buckets describe "gs://${STATE_BUCKET}" >/dev/null 2>&1; then
  gcloud storage buckets create "gs://${STATE_BUCKET}" \
    --project "$PROJECT_ID" \
    --location "$REGION" \
    --uniform-bucket-level-access
fi

gcloud storage buckets update "gs://${STATE_BUCKET}" \
  --versioning \
  --public-access-prevention

gcloud storage buckets add-iam-policy-binding "gs://${STATE_BUCKET}" \
  --member="serviceAccount:${INFRA_SA}" \
  --role="roles/storage.objectAdmin" >/dev/null

gcloud storage buckets add-iam-policy-binding "gs://${STATE_BUCKET}" \
  --member="serviceAccount:${APPLY_SA}" \
  --role="roles/storage.objectAdmin" >/dev/null

ensure_workload_identity_provider() {
  local pool_id="$1"
  local provider_id="$2"
  local display_name="$3"
  local attribute_mapping="$4"
  local attribute_condition="$5"

  if ! gcloud iam workload-identity-pools describe "$pool_id" \
    --location=global >/dev/null 2>&1; then
    gcloud iam workload-identity-pools create "$pool_id" \
      --location=global \
      --display-name="$display_name"
  fi

  if gcloud iam workload-identity-pools providers describe "$provider_id" \
    --workload-identity-pool="$pool_id" \
    --location=global >/dev/null 2>&1; then
    gcloud iam workload-identity-pools providers update-oidc "$provider_id" \
      --workload-identity-pool="$pool_id" \
      --location=global \
      --issuer-uri="https://token.actions.githubusercontent.com" \
      --attribute-mapping="$attribute_mapping" \
      --attribute-condition="$attribute_condition"
  else
    gcloud iam workload-identity-pools providers create-oidc "$provider_id" \
      --workload-identity-pool="$pool_id" \
      --location=global \
      --display-name="$display_name" \
      --issuer-uri="https://token.actions.githubusercontent.com" \
      --attribute-mapping="$attribute_mapping" \
      --attribute-condition="$attribute_condition"
  fi
}

ATTRIBUTE_MAPPING="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_id=assertion.repository_id,attribute.repository_owner_id=assertion.repository_owner_id,attribute.ref=assertion.ref,attribute.environment=assertion.environment"
PLAN_ATTRIBUTE_CONDITION="assertion.repository_id=='${REPO_ID}' && assertion.repository_owner_id=='${OWNER_ID}' && assertion.ref=='refs/heads/main' && assertion.environment=='production'"
APPLY_ATTRIBUTE_MAPPING="${ATTRIBUTE_MAPPING},attribute.job_workflow_ref=assertion.job_workflow_ref"
APPLY_ATTRIBUTE_CONDITION="${PLAN_ATTRIBUTE_CONDITION} && assertion.job_workflow_ref=='${APPLY_WORKFLOW_REF}'"

ensure_workload_identity_provider \
  "$POOL_ID" \
  "$PROVIDER_ID" \
  "RentStage production plan" \
  "$ATTRIBUTE_MAPPING" \
  "$PLAN_ATTRIBUTE_CONDITION"

ensure_workload_identity_provider \
  "$APPLY_POOL_ID" \
  "$APPLY_PROVIDER_ID" \
  "RentStage production apply" \
  "$APPLY_ATTRIBUTE_MAPPING" \
  "$APPLY_ATTRIBUTE_CONDITION"

PROVIDER="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_ID}/providers/${PROVIDER_ID}"
APPLY_PROVIDER="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${APPLY_POOL_ID}/providers/${APPLY_PROVIDER_ID}"
PRINCIPAL="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_ID}/attribute.repository_id/${REPO_ID}"
APPLY_PRINCIPAL="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${APPLY_POOL_ID}/attribute.repository_id/${REPO_ID}"

gcloud iam service-accounts add-iam-policy-binding "$INFRA_SA" \
  --member="$PRINCIPAL" \
  --role=roles/iam.workloadIdentityUser \
  --quiet >/dev/null

gcloud iam service-accounts add-iam-policy-binding "$APPLY_SA" \
  --member="$APPLY_PRINCIPAL" \
  --role=roles/iam.workloadIdentityUser \
  --quiet >/dev/null

cat <<EOF

Production bootstrap complete. No service-account key was created.

Create a protected GitHub Environment named: production
- Restrict deployment branches to: main
- Add a required reviewer
- Do not enable production deployment yet

Environment variables:
GCP_PROJECT_ID=${PROJECT_ID}
GCP_REGION=${REGION}
GCP_WIF_PROVIDER=${PROVIDER}
GCP_INFRA_SERVICE_ACCOUNT=${INFRA_SA}
GCP_INFRA_APPLY_WIF_PROVIDER=${APPLY_PROVIDER}
GCP_INFRA_APPLY_SERVICE_ACCOUNT=${APPLY_SA}
GCP_DEPLOY_SERVICE_ACCOUNT=${DEPLOY_SA}
GCP_ARTIFACT_REPOSITORY=rentstage
TF_STATE_BUCKET=${STATE_BUCKET}

Environment secret:
PRODUCTION_DATABASE_PASSWORD=<24-64 random alphanumeric characters>

Repository variable:
PRODUCTION_INFRA_APPLY_ENABLED=false
PRODUCTION_DEPLOY_ENABLED=false

The production identities are bound to repository ID ${REPO_ID}, owner ID ${OWNER_ID},
refs/heads/main, and the protected production GitHub Environment. The apply
identity additionally accepts tokens only from ${APPLY_WORKFLOW_REF}.

The plan identity is read-only in the project except for the isolated state
bucket. The apply identity can access the state bucket but receives no project
mutation roles until scripts/gcp/production-apply-access.sh grants them just in
time. The reserved deployment identity has no project roles and no WIF
impersonation binding in this release.

Next safe action: run Production Infrastructure Plan. Keep both production
gates false until the plan, monthly cost, and v0.17.1 apply procedure are approved.
EOF
