#!/usr/bin/env bash
set -euo pipefail
: "${GCP_PROJECT_ID:?required}"
: "${GCP_REGION:?required}"
API_SERVICE="${API_SERVICE:-rentstage-api-staging}"
WEB_SERVICE="${WEB_SERVICE:-rentstage-web-staging}"
PREVIOUS_API_REVISION="${PREVIOUS_API_REVISION:-__NONE__}"
PREVIOUS_WEB_REVISION="${PREVIOUS_WEB_REVISION:-__NONE__}"
rollback_service() {
  local service="$1" revision="$2"
  if [[ "$revision" == "__NONE__" || -z "$revision" ]]; then
    gcloud run services delete "$service" --project "$GCP_PROJECT_ID" --region "$GCP_REGION" --quiet || true
  else
    gcloud run services update-traffic "$service" --project "$GCP_PROJECT_ID" --region "$GCP_REGION" --to-revisions "${revision}=100" --quiet || true
  fi
}
rollback_service "$WEB_SERVICE" "$PREVIOUS_WEB_REVISION"
rollback_service "$API_SERVICE" "$PREVIOUS_API_REVISION"
