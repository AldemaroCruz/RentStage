# Upgrade RentStage v0.19.1 → v0.20.0

RentStage v0.20.0 adds AI-assisted web-chat response drafts while preserving the existing public-session API and mandatory human-publication boundary.

The default mode remains deterministic rules. Vertex AI is optional and must be enabled explicitly. No database migration, public endpoint change, Meta credential, telephone number, or automatic customer delivery is introduced.

## 1. Compatibility summary

- Database migration ordering remains at `016_omnichannel_web_chat.sql`.
- Existing web-chat sessions and hashed session tokens remain compatible.
- Existing public request and response shapes remain unchanged.
- Drafts remain private until an authorized team member publishes them.
- `ASSISTANT_AI_MODE=rules` requires no cloud credential or external call.
- The web application now uses the committed `package-lock.json` through `npm ci`.

## 2. Runtime configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `ASSISTANT_AI_MODE` | `rules` | Selects `rules` or `vertex`. |
| `ASSISTANT_AI_PROJECT_ID` | empty | Required when mode is `vertex`. |
| `ASSISTANT_AI_LOCATION` | `us-central1` | Vertex AI regional endpoint. |
| `ASSISTANT_AI_MODEL` | `gemini-2.5-flash` | Vertex AI model identifier. |
| `ASSISTANT_AI_TIMEOUT` | `8s` | Provider timeout; accepted range is 1–20 seconds. |
| `ASSISTANT_AI_MAX_OUTPUT_TOKENS` | `512` | Output limit; accepted range is 64–2048 tokens. |

Invalid modes, missing Vertex fields, and values outside the timeout or token bounds fail during application configuration validation.

## 3. Safe default upgrade

No AI credential is required for the default upgrade:

```dotenv
ASSISTANT_AI_MODE=rules
ASSISTANT_AI_PROJECT_ID=
ASSISTANT_AI_LOCATION=us-central1
ASSISTANT_AI_MODEL=gemini-2.5-flash
ASSISTANT_AI_TIMEOUT=8s
ASSISTANT_AI_MAX_OUTPUT_TOKENS=512
```

Rebuild the services:

```powershell
docker compose up -d --build
docker compose ps
Invoke-RestMethod -Uri "http://127.0.0.1:8080/healthz"
```

## 4. Optional local Vertex AI setup

Prefer Application Default Credentials with service-account impersonation. Do not create or commit a long-lived JSON service-account key.

Required Google Cloud setup:

1. Enable `aiplatform.googleapis.com` and `iamcredentials.googleapis.com`.
2. Grant the runtime or impersonated service account `roles/aiplatform.user`.
3. Grant `roles/serviceusage.serviceUsageConsumer` when required by the project quota policy.
4. Grant the developer `roles/iam.serviceAccountTokenCreator` on the impersonated service account.
5. Create local ADC:

```powershell
gcloud auth application-default login `
  --impersonate-service-account=SERVICE_ACCOUNT_EMAIL
```

For Docker Compose, mount the host ADC file read-only into the API container and set:

```dotenv
ASSISTANT_AI_MODE=vertex
ASSISTANT_AI_PROJECT_ID=YOUR_PROJECT_ID
ASSISTANT_AI_LOCATION=us-central1
ASSISTANT_AI_MODEL=gemini-2.5-flash
GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/adc.json
```

Keep the local Compose override outside the repository and remove it after testing. Return the API to `rules` mode after the Vertex smoke test.

## 5. Staging activation

The deployment workflow recognizes protected repository variables:

- `STAGING_ASSISTANT_AI_MODE` — defaults to `rules`;
- `STAGING_ASSISTANT_AI_LOCATION` — defaults to the configured GCP region;
- `STAGING_ASSISTANT_AI_MODEL` — defaults to `gemini-2.5-flash`.

When Vertex mode is enabled, the staging project ID is supplied from the existing GCP deployment configuration. Terraform enables the Vertex AI API and grants the API runtime service account `roles/aiplatform.user`.

Cloud Run should use its attached runtime service account through ADC. Do not mount a developer credential or JSON key in staging.

## 6. Source validation

From the repository root:

```powershell
bash scripts/ci/check-version.sh
python .\scripts\ci\check-workflow-yaml.py
python .\scripts\ci\check-migrations.py
bash scripts/ci/check-sensitive-files.sh

docker run --rm `
  -v "${PWD}:/workspace" `
  -w /workspace/apps/api `
  golang:1.26.6-alpine `
  sh -c 'test -z "$(gofmt -l .)" && go mod verify && go test ./... && go vet ./...'

docker run --rm `
  -v "${PWD}:/workspace" `
  -v "rentstage-web-v0200-modules:/workspace/apps/web/node_modules" `
  -w /workspace/apps/web `
  node:24-alpine `
  sh -c "npm ci --no-audit --no-fund && npm run test:ci"

docker compose config *> $null
git diff --check
```

## 7. Integration validation

```powershell
docker compose up -d --build
docker compose ps

powershell -NoLogo -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\run-smoke-suite.ps1
```

Validate one deterministic draft before enabling Vertex. Then validate one Vertex initial draft and one idempotent follow-up retry. Confirm that the visitor sees no outbound response until an authorized team member publishes it.

## 8. Rollback

Application rollback does not require a schema rollback because v0.20.0 adds no migration.

To disable external AI generation immediately, set:

```dotenv
ASSISTANT_AI_MODE=rules
```

Recreate the API service. Existing conversations, drafts, published messages, and session tokens remain compatible.
