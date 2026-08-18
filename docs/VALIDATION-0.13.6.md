# RentStage v0.13.6 validation record

## Release contracts

- Incremental update from the deployed v0.13.5 staging baseline.
- No SQL migration, API contract, Terraform resource, environment variable, or tenant-data change.
- Local Docker Compose continues to use the Firebase Authentication emulator and documented owner account.
- Non-local web builds must not prefill or render local demo credentials.
- DTE remains MOCK / TEST in automated staging validation.

## Focused frontend checks

The runtime-configuration unit tests verify:

- an omitted or explicit `true` emulator value preserves local development;
- `false` disables local authentication UI;
- malformed non-local values fail closed;
- the local account is defined once instead of being duplicated across the login page.

The production build must be exercised with both supported modes:

```text
NEXT_PUBLIC_USE_AUTH_EMULATOR=true   local Docker Compose
NEXT_PUBLIC_USE_AUTH_EMULATOR=false  GCP staging
```

For the staging build, the generated login page must start with empty email and password fields and must not contain `RentStage123!`.

## Repository validation

```bash
bash scripts/ci/check-version.sh
python scripts/ci/check-workflow-yaml.py
python scripts/ci/check-migrations.py
docker compose config >/dev/null
```

Frontend validation:

```bash
cd apps/web
npm install --no-audit --no-fund
npm run test:ci
```

The existing CI/CD pipeline additionally runs Go unit/race/vet checks, Terraform validation, source and dependency security, Docker integration, image scans, the complete local smoke suite, staging deployment, and the remote smoke suite.

## Manual acceptance

1. Open the local login page and confirm the local owner account remains visible and prefilled.
2. Open the staging login page and confirm both fields are empty and no local credential card is rendered.
3. Sign in to staging with the `STAGING_SMOKE_EMAIL` account and its rotated secret.
4. Confirm the dashboard release strip displays `v0.13.6`, the connected product scope, and DTE MOCK / TEST wording.
5. Confirm the dashboard strip wraps without horizontal overflow at mobile width.
