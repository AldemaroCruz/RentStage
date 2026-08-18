#!/usr/bin/env bash
set -euo pipefail
required=(GCP_PROJECT_ID GCP_REGION API_IMAGE WEB_IMAGE CLOUD_SQL_CONNECTION RENTSTAGE_VERSION)
for name in "${required[@]}"; do
  [[ -n "${!name:-}" ]] || { echo "$name is required" >&2; exit 1; }
done
API_SERVICE="${API_SERVICE:-rentstage-api-staging}"
WEB_SERVICE="${WEB_SERVICE:-rentstage-web-staging}"
API_RUNTIME_SA="${API_RUNTIME_SA:-rentstage-api-stg@${GCP_PROJECT_ID}.iam.gserviceaccount.com}"
WEB_RUNTIME_SA="${WEB_RUNTIME_SA:-rentstage-web-stg@${GCP_PROJECT_ID}.iam.gserviceaccount.com}"
SEED_DEMO_DATA="${STAGING_SEED_DEMO_DATA:-true}"
ALLOW_DEMO="false"
[[ "$SEED_DEMO_DATA" == "true" ]] && ALLOW_DEMO="true"
PLACEHOLDER_ORIGIN="https://staging.invalid"

echo "attempted=true" >> "${GITHUB_OUTPUT:-/dev/null}"
gcloud config set project "$GCP_PROJECT_ID" >/dev/null

echo "Deploying the private RentStage API..."
gcloud run deploy "$API_SERVICE" \
  --project "$GCP_PROJECT_ID" \
  --region "$GCP_REGION" \
  --platform managed \
  --image "$API_IMAGE" \
  --service-account "$API_RUNTIME_SA" \
  --no-allow-unauthenticated \
  --ingress all \
  --port 8080 \
  --cpu 1 \
  --memory 512Mi \
  --min-instances 0 \
  --max-instances 3 \
  --concurrency 40 \
  --timeout 300 \
  --add-cloudsql-instances "$CLOUD_SQL_CONNECTION" \
  --set-secrets "DATABASE_URL=rentstage-staging-database-url:latest,PUBLIC_REQUEST_FINGERPRINT_SALT=rentstage-staging-fingerprint-salt:latest" \
  --set-env-vars "APP_ENV=staging,FIREBASE_PROJECT_ID=${GCP_PROJECT_ID},GCLOUD_PROJECT=${GCP_PROJECT_ID},SEED_DEMO_DATA=${SEED_DEMO_DATA},ALLOW_DEMO_DATA_OUTSIDE_LOCAL=${ALLOW_DEMO},SESSION_DURATION=12h,SESSION_COOKIE_NAME=rentstage_session,TENANT_COOKIE_NAME=rentstage_tenant,CSRF_COOKIE_NAME=rentstage_csrf,COOKIE_SECURE=true,REQUIRE_VERIFIED_EMAIL=false,LOCAL_AUTH_BOOTSTRAP=false,WEB_BASE_URL=${PLACEHOLDER_ORIGIN},CORS_ALLOWED_ORIGINS=${PLACEHOLDER_ORIGIN}" \
  --quiet

API_URL="$(gcloud run services describe "$API_SERVICE" --region "$GCP_REGION" --format='value(status.url)')"
[[ -n "$API_URL" ]] || { echo "Unable to resolve API service URL" >&2; exit 1; }

gcloud run services remove-iam-policy-binding "$API_SERVICE" \
  --region "$GCP_REGION" --member='allUsers' --role='roles/run.invoker' --quiet >/dev/null 2>&1 || true
gcloud run services add-iam-policy-binding "$API_SERVICE" \
  --region "$GCP_REGION" \
  --member="serviceAccount:${WEB_RUNTIME_SA}" \
  --role='roles/run.invoker' \
  --quiet >/dev/null

echo "Deploying the public RentStage web service..."
gcloud run deploy "$WEB_SERVICE" \
  --project "$GCP_PROJECT_ID" \
  --region "$GCP_REGION" \
  --platform managed \
  --image "$WEB_IMAGE" \
  --service-account "$WEB_RUNTIME_SA" \
  --no-invoker-iam-check \
  --ingress all \
  --port 3000 \
  --cpu 1 \
  --memory 512Mi \
  --min-instances 0 \
  --max-instances 3 \
  --concurrency 80 \
  --timeout 300 \
  --set-env-vars "API_INTERNAL_URL=${API_URL},API_AUDIENCE=${API_URL},CLOUD_RUN_IDENTITY_TOKEN_ENABLED=true,RENTSTAGE_VERSION=${RENTSTAGE_VERSION}" \
  --quiet

WEB_URL="$(gcloud run services describe "$WEB_SERVICE" --region "$GCP_REGION" --format='value(status.url)')"
[[ -n "$WEB_URL" ]] || { echo "Unable to resolve web service URL" >&2; exit 1; }
WEB_HOST="${WEB_URL#https://}"

# Identity Platform must trust the generated Cloud Run hostname. Preserve every
# existing authorized domain and add the exact staging hostname idempotently.
identity_config_url="https://identitytoolkit.googleapis.com/admin/v2/projects/${GCP_PROJECT_ID}/config"
identity_token="$(gcloud auth print-access-token)"
identity_config="$(mktemp)"
identity_patch="$(mktemp)"
trap 'rm -f "$identity_config" "$identity_patch"' EXIT
curl --fail --silent --show-error \
  -H "Authorization: Bearer ${identity_token}" \
  "$identity_config_url" > "$identity_config"
python3 - "$identity_config" "$identity_patch" "$WEB_HOST" <<'PY_AUTH_DOMAINS'
import json
import sys
source, destination, host = sys.argv[1:]
with open(source, encoding="utf-8") as handle:
    config = json.load(handle)
domains = sorted(set(config.get("authorizedDomains", [])) | {host})
with open(destination, "w", encoding="utf-8") as handle:
    json.dump({"authorizedDomains": domains}, handle)
PY_AUTH_DOMAINS
curl --fail --silent --show-error \
  -X PATCH \
  -H "Authorization: Bearer ${identity_token}" \
  -H 'Content-Type: application/json' \
  --data-binary "@${identity_patch}" \
  "${identity_config_url}?updateMask=authorizedDomains" >/dev/null

echo "Updating API CORS and cookie origin to the real staging URL..."
gcloud run services update "$API_SERVICE" \
  --project "$GCP_PROJECT_ID" \
  --region "$GCP_REGION" \
  --update-env-vars "WEB_BASE_URL=${WEB_URL},CORS_ALLOWED_ORIGINS=${WEB_URL}" \
  --quiet >/dev/null

wait_for_http_200() {
  local name="$1"
  local url="$2"
  local attempts="$3"
  local delay_seconds="$4"
  local attempt status body_file

  body_file="$(mktemp)"

  for ((attempt = 1; attempt <= attempts; attempt++)); do
    status="$(
      curl --silent --show-error \
        --output "$body_file" \
        --write-out '%{http_code}' \
        "$url" || true
    )"

    if [[ "$status" == "200" ]]; then
      echo "${name} is ready."
      rm -f "$body_file"
      return 0
    fi

    echo "${name} returned HTTP ${status:-000} (attempt ${attempt}/${attempts})."

    if ((attempt < attempts)); then
      sleep "$delay_seconds"
    fi
  done

  echo "${name} did not become ready after ${attempts} attempts." >&2
  sed -n '1,20p' "$body_file" >&2 || true
  rm -f "$body_file"
  return 1
}

# Public access and new revisions can take a short time to become reachable.
wait_for_http_200 \
  "RentStage web" \
  "${WEB_URL}/api/healthz" \
  12 \
  5

# The API invoker binding is eventually consistent. On the first deployment,
# Cloud Run can temporarily reject the web runtime identity with HTTP 403.
wait_for_http_200 \
  "RentStage private API through the web proxy" \
  "${WEB_URL}/api/backend/readyz" \
  60 \
  10

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  {
    echo "api_url=${API_URL}"
    echo "web_url=${WEB_URL}"
  } >> "$GITHUB_OUTPUT"
fi
printf 'API_URL=%s\nWEB_URL=%s\n' "$API_URL" "$WEB_URL"
