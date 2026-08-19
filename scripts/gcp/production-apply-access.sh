#!/usr/bin/env bash
set -euo pipefail

OPERATION="${1:-status}"
: "${PROJECT_ID:?Set PROJECT_ID to the dedicated production project ID}"

APPLY_SA_ID="${APPLY_SA_ID:-rentstage-infra-apply-prod}"
APPLY_SA="${APPLY_SA_ID}@${PROJECT_ID}.iam.gserviceaccount.com"

APPLY_ROLES=(
  roles/artifactregistry.admin
  roles/cloudsql.admin
  roles/firebase.admin
  roles/iam.roleAdmin
  roles/iam.serviceAccountAdmin
  roles/identityplatform.admin
  roles/resourcemanager.projectIamAdmin
  roles/secretmanager.admin
  roles/serviceusage.apiKeysAdmin
  roles/serviceusage.serviceUsageAdmin
  roles/viewer
)

command -v gcloud >/dev/null || {
  echo "gcloud is required" >&2
  exit 1
}

case "$OPERATION" in
  grant|revoke|status) ;;
  *)
    echo "Usage: $0 grant|revoke|status" >&2
    exit 2
    ;;
esac

ACTIVE_PROJECT="$(gcloud config get-value project 2>/dev/null)"
if [[ "$ACTIVE_PROJECT" != "$PROJECT_ID" ]]; then
  echo "Active gcloud project is ${ACTIVE_PROJECT:-unset}; expected ${PROJECT_ID}." >&2
  exit 1
fi

gcloud iam service-accounts describe "$APPLY_SA" >/dev/null

has_role() {
  local role="$1"
  gcloud projects get-iam-policy "$PROJECT_ID" \
    --flatten='bindings[].members' \
    --filter="bindings.role=${role} AND bindings.members=serviceAccount:${APPLY_SA}" \
    --format='value(bindings.role)' | grep -Fxq "$role"
}

show_status() {
  echo "Production Terraform apply identity: ${APPLY_SA}"
  echo "Direct project roles:"
  gcloud projects get-iam-policy "$PROJECT_ID" \
    --flatten='bindings[].members' \
    --filter="bindings.members=serviceAccount:${APPLY_SA}" \
    --format='table(bindings.role)' || true
}

if [[ "$OPERATION" == "status" ]]; then
  show_status
  exit 0
fi

: "${CONFIRM_PROJECT_ID:?Set CONFIRM_PROJECT_ID to the same production project ID}"
if [[ "$CONFIRM_PROJECT_ID" != "$PROJECT_ID" ]]; then
  echo "CONFIRM_PROJECT_ID does not match PROJECT_ID." >&2
  exit 1
fi

if [[ "$OPERATION" == "grant" ]]; then
  BILLING_ENABLED="$(
    gcloud billing projects describe "$PROJECT_ID" \
      --format='value(billingEnabled)'
  )"
  if [[ "$BILLING_ENABLED" != "True" && "$BILLING_ENABLED" != "true" ]]; then
    echo "Project ${PROJECT_ID} does not have billing enabled." >&2
    exit 1
  fi

  echo "Granting temporary production infrastructure permissions..."
  for role in "${APPLY_ROLES[@]}"; do
    if ! has_role "$role"; then
      gcloud projects add-iam-policy-binding "$PROJECT_ID" \
        --member="serviceAccount:${APPLY_SA}" \
        --role="$role" \
        --condition=None \
        --quiet >/dev/null
    fi
  done
  echo "Temporary apply permissions granted. Revoke them after the reviewed workflow."
else
  echo "Revoking temporary production infrastructure permissions..."
  for role in "${APPLY_ROLES[@]}"; do
    if has_role "$role"; then
      gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
        --member="serviceAccount:${APPLY_SA}" \
        --role="$role" \
        --condition=None \
        --quiet >/dev/null
    fi
  done
  echo "Temporary apply permissions revoked."
fi

show_status
