# Upgrade RentStage v0.20.0 → v0.21.0

RentStage v0.21.0 extends private web-chat drafts with bounded conversation context, public-catalog grounding, reviewer evidence, a structured sales brief, and deterministic commercial-claim guardrails.

## Database

No migration is required. The latest migration remains:

```text
016_omnichannel_web_chat.sql
```

Grounding references, sales briefs, provenance, and fallback diagnostics use the existing `assistant_messages.metadata` JSONB column.

## Configuration

Existing v0.20.0 Vertex AI settings remain valid. The recommended timeout is now 20 seconds:

```dotenv
ASSISTANT_AI_MODE=rules
ASSISTANT_AI_LOCATION=us-central1
ASSISTANT_AI_MODEL=gemini-2.5-flash
ASSISTANT_AI_TIMEOUT=20s
ASSISTANT_AI_MAX_OUTPUT_TOKENS=512
```

Local development should remain in `rules` mode unless an explicit Vertex smoke test is running. Vertex mode continues to require `ASSISTANT_AI_PROJECT_ID` and Application Default Credentials. Do not commit ADC files or mount them into the web container.

No new secret, Terraform resource, API enablement, or IAM role is introduced beyond the v0.20.0 Vertex AI setup.

## Deployment behavior

- Existing public-chat request and response shapes are unchanged.
- Assistant drafts remain private until an authenticated user publishes them.
- Invalid provider output produces a deterministic fallback instead of failing the customer request.
- No quote, reservation, inventory, invoice, payment, DTE, or external message is created automatically.

## Verification

Before merging, run:

```bash
bash scripts/ci/check-version.sh
docker compose config >/dev/null
cd apps/api
test -z "$(gofmt -l .)"
go mod verify
go test -race -shuffle=on -count=1 ./...
go vet ./...
```

Then run `npm ci --no-audit --no-fund && npm run test:ci` in `apps/web`, rebuild the local stack, and execute `scripts/run-smoke-suite.ps1`.

## Rollback

Application rollback to v0.20.0 requires no schema rollback. Any v0.21.0 metadata fields remain harmless JSON values that older code ignores. Restore `rules` mode and remove temporary ADC mounts after every Vertex smoke test.
