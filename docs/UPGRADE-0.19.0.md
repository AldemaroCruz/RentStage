# Upgrade RentStage v0.18.1 → v0.19.0

v0.19.0 adds the omnichannel conversation foundation and a first-party web-chat MVP for the public catalog. The migration is additive and no new environment variable, cloud service, secret, IAM permission, Terraform apply, Meta account, or telephone number is required.

## 1. Source validation

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
  sh -c 'find . -type f -name "*.go" -exec gofmt -w {} + && test -z "$(gofmt -l .)" && go test ./...'

git diff --check
git status --short
```

## 2. Apply the additive migration and rebuild

```powershell
docker compose up -d --build api web
docker compose ps

docker compose exec -T db psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations WHERE version = '016_omnichannel_web_chat.sql';"
```

Expected result: API and web become healthy and migration `016_omnichannel_web_chat.sql` appears once.

## 3. Enable the tenant channel

Open `/settings/public-catalog`, keep the catalog published, enable **Web chat**, and save. The public catalog response must then report `settings.web_chat_enabled: true`.

## 4. End-to-end acceptance

1. Open `/p/<tenant-slug>` and expand **Need help?**.
2. Start a conversation with a name, optional email, message, and accepted contact notice.
3. Open `/assistant` and confirm that the conversation is labeled **Web chat** and **Human review**.
4. Confirm the prepared draft is not visible in the visitor widget.
5. Edit the draft if needed and press **Publish to web chat**.
6. Confirm that the exact approved response appears in the widget within four seconds without reloading.
7. Send a follow-up message and confirm that an idempotent retry creates no duplicate inbound message.

## Data compatibility

Migration `016_omnichannel_web_chat.sql` extends the existing provider constraints, adds the catalog feature switch, and creates tenant-scoped public session storage. Existing demo and WhatsApp conversations remain unchanged. Existing published catalogs default the chat switch to disabled unless explicitly enabled or seeded for the local demo.

## Rollback

Disable **Web chat** before rolling back application source. Existing public sessions immediately become unavailable. The additive migration and its rows can remain in place for a later forward deployment; do not drop the table or rewrite migration history. Meta local behavior, quote requests, quotes, reservations, and inventory remain independent.
