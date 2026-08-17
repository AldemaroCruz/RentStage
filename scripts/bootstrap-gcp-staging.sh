#!/usr/bin/env bash
set -euo pipefail
: "${PROJECT_ID:?Set PROJECT_ID to the billing-enabled Google Cloud project ID}"
: "${GITHUB_REPOSITORY:?Set GITHUB_REPOSITORY to owner/repository}"
REGION="${REGION:-us-east1}"
POOL_ID="${POOL_ID:-github-actions}"
PROVIDER_ID="${PROVIDER_ID:-rentstage-staging}"
INFRA_SA_ID="${INFRA_SA_ID:-rentstage-infra-github}"
DEPLOY_SA_ID="${DEPLOY_SA_ID:-rentstage-deploy-github}"
STATE_BUCKET="${STATE_BUCKET:-${PROJECT_ID}-rentstage-tfstate}"
command -v gcloud >/dev/null || { echo "gcloud is required" >&2; exit 1; }
command -v gh >/dev/null || { echo "GitHub CLI is required and must be authenticated" >&2; exit 1; }

REPO_ID="$(gh api "repos/${GITHUB_REPOSITORY}" --jq '.id')"
OWNER_ID="$(gh api "repos/${GITHUB_REPOSITORY}" --jq '.owner.id')"
[[ "$REPO_ID" =~ ^[0-9]+$ && "$OWNER_ID" =~ ^[0-9]+$ ]] || { echo "Unable to resolve immutable GitHub IDs" >&2; exit 1; }

gcloud config set project "$PROJECT_ID" >/dev/null
gcloud services enable \
  serviceusage.googleapis.com cloudresourcemanager.googleapis.com iam.googleapis.com \
  iamcredentials.googleapis.com sts.googleapis.com storage.googleapis.com --quiet
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"

ensure_sa() {
  local id="$1" name="$2"
  gcloud iam service-accounts describe "${id}@${PROJECT_ID}.iam.gserviceaccount.com" >/dev/null 2>&1 \
    || gcloud iam service-accounts create "$id" --display-name "$name"
}
ensure_sa "$INFRA_SA_ID" "RentStage GitHub infrastructure"
ensure_sa "$DEPLOY_SA_ID" "RentStage GitHub deployment"
INFRA_SA="${INFRA_SA_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
DEPLOY_SA="${DEPLOY_SA_ID}@${PROJECT_ID}.iam.gserviceaccount.com"

for role in \
  roles/artifactregistry.admin roles/cloudsql.admin roles/secretmanager.admin \
  roles/resourcemanager.projectIamAdmin roles/iam.serviceAccountAdmin \
  roles/serviceusage.serviceUsageAdmin roles/firebase.admin roles/identityplatform.admin \
  roles/serviceusage.apiKeysAdmin roles/storage.admin; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" --member="serviceAccount:${INFRA_SA}" --role="$role" --condition=None --quiet >/dev/null
done

if ! gcloud storage buckets describe "gs://${STATE_BUCKET}" >/dev/null 2>&1; then
  gcloud storage buckets create "gs://${STATE_BUCKET}" --project "$PROJECT_ID" --location "$REGION" --uniform-bucket-level-access
fi
gcloud storage buckets update "gs://${STATE_BUCKET}" --versioning --public-access-prevention
gcloud storage buckets add-iam-policy-binding "gs://${STATE_BUCKET}" --member="serviceAccount:${INFRA_SA}" --role="roles/storage.admin" >/dev/null

if ! gcloud iam workload-identity-pools describe "$POOL_ID" --location=global >/dev/null 2>&1; then
  gcloud iam workload-identity-pools create "$POOL_ID" --location=global --display-name="GitHub Actions"
fi
MAPPING="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_id=assertion.repository_id,attribute.repository_owner_id=assertion.repository_owner_id,attribute.ref=assertion.ref,attribute.environment=assertion.environment"
CONDITION="assertion.repository_id=='${REPO_ID}' && assertion.repository_owner_id=='${OWNER_ID}' && assertion.ref=='refs/heads/main' && assertion.environment=='staging'"
if gcloud iam workload-identity-pools providers describe "$PROVIDER_ID" --workload-identity-pool="$POOL_ID" --location=global >/dev/null 2>&1; then
  gcloud iam workload-identity-pools providers update-oidc "$PROVIDER_ID" --workload-identity-pool="$POOL_ID" --location=global --issuer-uri="https://token.actions.githubusercontent.com" --attribute-mapping="$MAPPING" --attribute-condition="$CONDITION"
else
  gcloud iam workload-identity-pools providers create-oidc "$PROVIDER_ID" --workload-identity-pool="$POOL_ID" --location=global --display-name="RentStage staging GitHub" --issuer-uri="https://token.actions.githubusercontent.com" --attribute-mapping="$MAPPING" --attribute-condition="$CONDITION"
fi
PROVIDER="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_ID}/providers/${PROVIDER_ID}"
PRINCIPAL="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_ID}/attribute.repository_id/${REPO_ID}"
for sa in "$INFRA_SA" "$DEPLOY_SA"; do
  gcloud iam service-accounts add-iam-policy-binding "$sa" --member="$PRINCIPAL" --role=roles/iam.workloadIdentityUser --quiet >/dev/null
done

cat <<EOF

Bootstrap complete. Create a GitHub Environment named: staging
Restrict deployment branches to: main
Add an approval reviewer before staging deploys if desired.

Environment variables:
GCP_PROJECT_ID=${PROJECT_ID}
GCP_REGION=${REGION}
GCP_WIF_PROVIDER=${PROVIDER}
GCP_INFRA_SERVICE_ACCOUNT=${INFRA_SA}
GCP_DEPLOY_SERVICE_ACCOUNT=${DEPLOY_SA}
GCP_ARTIFACT_REPOSITORY=rentstage
TF_STATE_BUCKET=${STATE_BUCKET}
STAGING_SEED_DEMO_DATA=true
RUN_FULL_STAGING_SMOKE=true

Environment secrets (choose strong values):
STAGING_DATABASE_PASSWORD=<24-64 random alphanumeric characters>
STAGING_SMOKE_EMAIL=owner@rentstage.local
STAGING_SMOKE_PASSWORD=<at least 12 characters>

Repository variables for optional GitHub paid security features:
ENABLE_CODEQL=false
ENABLE_DEPENDENCY_REVIEW=false

The WIF condition is bound to repository ID ${REPO_ID}, owner ID ${OWNER_ID}, refs/heads/main, and the staging environment.
Allow several minutes for IAM federation changes to propagate before the first workflow.
EOF
