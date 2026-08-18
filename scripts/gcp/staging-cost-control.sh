#!/usr/bin/env bash
set -euo pipefail

operation="${1:-status}"
: "${GCP_PROJECT_ID:?GCP_PROJECT_ID is required}"
: "${GCP_REGION:?GCP_REGION is required}"

sql_instance="${STAGING_SQL_INSTANCE:-rentstage-staging-postgres}"
deploy_gate="${STAGING_DEPLOY_ENABLED:-unset}"
confirmed="${CONFIRM_STAGING_COST_OPERATION:-false}"
cloud_run_services=(rentstage-web-staging rentstage-api-staging)

case "$operation" in
  status|pause|resume) ;;
  *)
    echo "Unsupported operation: $operation" >&2
    echo 'Expected status, pause, or resume.' >&2
    exit 2
    ;;
esac

if [[ "$operation" != "status" && "$confirmed" != "true" ]]; then
  echo 'Pause and resume require CONFIRM_STAGING_COST_OPERATION=true.' >&2
  exit 2
fi

if [[ "$operation" == "pause" && "$deploy_gate" != "false" ]]; then
  echo 'Refusing to pause Cloud SQL while STAGING_DEPLOY_ENABLED is not false.' >&2
  echo 'Disable the repository deployment gate first.' >&2
  exit 2
fi

sql_value() {
  local field="$1"
  gcloud sql instances describe "$sql_instance" \
    --project "$GCP_PROJECT_ID" \
    --format="value(${field})"
}

read_sql_status() {
  sql_state="$(sql_value state)"
  sql_policy="$(sql_value settings.activationPolicy)"
  [[ -n "$sql_state" ]] || sql_state='UNKNOWN'
  [[ -n "$sql_policy" ]] || sql_policy='UNKNOWN'
}

print_status() {
  local service details

  read_sql_status
  echo "Cloud SQL instance: $sql_instance"
  echo "Cloud SQL state: $sql_state"
  echo "Cloud SQL activation policy: $sql_policy"
  echo "Automatic staging deployment gate: $deploy_gate"
  echo
  echo 'Cloud Run services (configured to scale to zero):'

  for service in "${cloud_run_services[@]}"; do
    if details="$(
      gcloud run services describe "$service" \
        --project "$GCP_PROJECT_ID" \
        --region "$GCP_REGION" \
        --format='value(status.url,status.latestReadyRevisionName)' \
        2>/dev/null
    )"; then
      echo "  $service: $details"
    else
      echo "  $service: not deployed"
    fi
  done
}

write_summary() {
  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0

  {
    echo '## RentStage staging cost control'
    echo
    echo "- Operation: \`$operation\`"
    echo "- Cloud SQL instance: \`$sql_instance\`"
    echo "- Cloud SQL state: \`$sql_state\`"
    echo "- Activation policy: \`$sql_policy\`"
    echo "- Automatic deployment gate: \`$deploy_gate\`"
    echo '- Cloud Run: retained and configured with zero minimum instances'
    echo
    if [[ "$sql_policy" == 'NEVER' ]]; then
      echo 'Staging application operations that require the database are unavailable. Cloud SQL instance charges are suspended; storage, backups, networking, and other retained resources may still incur charges.'
    else
      echo 'Staging database activation is enabled. Normal instance charges may apply.'
    fi
  } >> "$GITHUB_STEP_SUMMARY"
}

case "$operation" in
  pause)
    echo "Pausing Cloud SQL instance $sql_instance..."
    gcloud sql instances patch "$sql_instance" \
      --project "$GCP_PROJECT_ID" \
      --activation-policy=NEVER \
      --quiet
    read_sql_status
    if [[ "$sql_policy" != 'NEVER' ]]; then
      echo "Cloud SQL activation policy is $sql_policy; expected NEVER." >&2
      exit 1
    fi
    echo 'Cloud SQL is paused. Cloud Run services remain deployed and scale to zero.'
    ;;
  resume)
    echo "Resuming Cloud SQL instance $sql_instance..."
    gcloud sql instances patch "$sql_instance" \
      --project "$GCP_PROJECT_ID" \
      --activation-policy=ALWAYS \
      --quiet
    read_sql_status
    if [[ "$sql_policy" != 'ALWAYS' ]]; then
      echo "Cloud SQL activation policy is $sql_policy; expected ALWAYS." >&2
      exit 1
    fi
    echo 'Cloud SQL activation is enabled. Re-enable the deployment gate only when staging is ready.'
    ;;
  status)
    read_sql_status
    ;;
esac

echo
print_status
write_summary
