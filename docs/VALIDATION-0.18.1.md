# Validation for RentStage v0.18.1

Expected checks:

```text
bash scripts/ci/check-version.sh
python scripts/ci/check-workflow-yaml.py
python scripts/ci/check-migrations.py
bash scripts/ci/check-sensitive-files.sh
cd apps/api && gofmt -w . && go test ./...
cd apps/web && npm install --no-audit --no-fund --package-lock=false && npm run test:ci
docker compose config
pwsh -File scripts/test-meta-local.ps1
```

Acceptance criteria:

- Version contracts report `0.18.1`.
- Migration ordering ends at `015_meta_application_readiness.sql`.
- Opt-out, delivery status, readiness, and legal-contact tests pass.
- Local Meta contract remains loopback-only.
- Cloud outbound is rejected and no secret value appears in source, logs, database migration, or readiness responses.
