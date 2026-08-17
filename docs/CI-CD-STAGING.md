# RentStage CI/CD and GCP staging

## Environment policy

| Environment | Runtime | Purpose |
|---|---|---|
| Development | Docker Compose on localhost | Daily coding and full smoke suite |
| Staging | Google Cloud Run + Cloud SQL | Closed-beta validation |
| Production | Not created | Deferred |

`main` is the staging release branch. Pull requests never authenticate to
Google Cloud.

## GitHub Actions

### `RentStage CI`

Every pull request and push to `main` executes:

1. Go module verification.
2. All Go unit tests with shuffle, race detector and coverage.
3. `go vet`, Govulncheck and high-severity Gosec.
4. Frontend install, optional lint/type/unit scripts and `next build`.
5. `npm audit` for high-severity production dependency findings.
6. Migration and repository security invariants.
7. Gitleaks and dependency review.
8. API/web container builds and Trivy scans.
9. An ephemeral Docker Compose environment.
10. Every existing RentStage PowerShell smoke test:
   authentication, packages, public catalog, Quote Portal, billing and DTE.

CI removes its volumes after every run.

### `RentStage Deep Security`

Runs weekly, manually, and for relevant pull requests:

- Optional CodeQL for Go and JavaScript/TypeScript when `ENABLE_CODEQL=true` and GitHub Code Security is available.
- Trivy filesystem, dependency, secret and configuration scanning.

### `RentStage Deploy Staging`

Runs only after a successful `RentStage CI` workflow on `main`, or manually
through the protected GitHub `staging` environment.

It authenticates with Google Cloud Workload Identity Federation, builds images
with provenance/SBOM, pushes commit-addressed tags, deploys API and web to
Cloud Run, and executes a non-destructive staging smoke test.

## One-time Google Cloud bootstrap

Prerequisites:

```text
Google Cloud CLI authenticated
A Google Cloud billing account
GitHub CLI authenticated (optional)
An existing GitHub repository owner/name
```

From PowerShell:

```powershell
.\scripts\gcp\bootstrap-staging.ps1 `
  -ProjectId "your-rentstage-staging-project" `
  -BillingAccount "000000-000000-000000" `
  -GitHubRepository "owner/rentstage" `
  -Region "us-central1" `
  -ConfigureGitHub
```

The script creates/enables:

```text
Artifact Registry
Cloud SQL PostgreSQL
rentstage database/user
Secret Manager values
API and web runtime service accounts
GitHub deployer service account
OIDC workload identity pool/provider
Required IAM bindings
```

The Cloud SQL database version is parameterized and defaults to
`POSTGRES_17`; override it only with a version supported in the selected
region.

## Firebase / Identity Platform

In the same staging project:

1. Enable Identity Platform.
2. Enable Email/Password authentication.
3. Create a web application.
4. Copy its project ID, API key and auth domain.
5. Create a staging smoke user. For the seeded AudioPro workspace, use the
   email expected by the demo seed or update the seed before closed beta.
6. Keep `REQUIRE_VERIFIED_EMAIL=false` only until the initial smoke account is
   established; then set it to true and verify users.

Create the initial user from PowerShell:

```powershell
$password = Read-Host "Staging smoke password" -AsSecureString
.\scripts\gcp\create-staging-smoke-user.ps1 `
  -FirebaseApiKey "YOUR_FIREBASE_WEB_API_KEY" `
  -Password $password `
  -GitHubRepository "owner/rentstage" `
  -ConfigureGitHub
```

## GitHub environment

Create the environment:

```text
staging
```

Recommended protection:

```text
Deployment branch: main only
Required reviewer: repository owner
Prevent self-review: enabled when a team is available
```

Required environment variables:

```text
GCP_PROJECT_ID
GCP_REGION
GCP_ARTIFACT_REPOSITORY
GCP_API_SERVICE
GCP_WEB_SERVICE
GCP_API_SERVICE_ACCOUNT
GCP_WEB_SERVICE_ACCOUNT
GCP_CLOUD_SQL_CONNECTION_NAME
GCP_WIF_PROVIDER
GCP_DEPLOYER_SERVICE_ACCOUNT
FIREBASE_PROJECT_ID
FIREBASE_API_KEY
FIREBASE_AUTH_DOMAIN
STAGING_PUBLIC_CATALOG_SLUG   optional
```

Optional authenticated-smoke secrets:

```text
STAGING_SMOKE_EMAIL
STAGING_SMOKE_PASSWORD
```

No GCP credential JSON belongs in GitHub.

## Initial deployment

1. Push v0.13.0 to a feature branch.
2. Open a pull request and let all CI jobs pass.
3. Merge to `main`.
4. Approve the `staging` environment deployment.
5. Read the API/web URLs from the workflow summary.
6. Open `/login`, `/settings/*`, the public catalog and Quote Portal manually.

## Rollback

Images are tagged with the immutable commit SHA. To roll back API:

```powershell
gcloud run deploy rentstage-api-staging `
  --project YOUR_PROJECT `
  --region YOUR_REGION `
  --image "YOUR_REGION-docker.pkg.dev/YOUR_PROJECT/rentstage/rentstage-api@KNOWN_DIGEST"
```

Repeat for web.

Database migrations are forward-only. Before a release containing a migration,
use the Cloud SQL automated backup and verify the restore plan. Do not attempt
to roll a database backward by deleting migration records.

## Local developer workflow

Nothing changes:

```powershell
docker compose build api web
docker compose up -d
powershell -ExecutionPolicy Bypass -File .\scripts\ci\run-smoke-suite.ps1
```

## v0.13.1 dependency and PowerShell troubleshooting

Before the first v0.13 pull request, regenerate and commit the Go module graph:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\ci\sync-go-modules.ps1
git add .\apps\api\go.mod .\apps\api\go.sum
git commit -m "build: synchronize Go module metadata"
```

The Docker-backed command uses Go 1.26.6, so a host Go installation is not
required. CI runs `go mod tidy` and rejects an uncommitted diff.

The local smoke wrapper supports either executable:

```text
pwsh        PowerShell 7
powershell  Windows PowerShell 5.1 fallback
```

Run it on Windows with:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\ci\run-smoke-suite.ps1
```
