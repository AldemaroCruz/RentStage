# RentStage v0.13.0 — GitHub Actions and GCP staging

## Scope

v0.13.0 creates one remote environment only:

```text
Local development
  Docker Compose + PostgreSQL + Firebase Auth Emulator

Staging
  GitHub Actions -> Artifact Registry -> Cloud Run
                                |-> Cloud SQL PostgreSQL
                                |-> Identity Platform
                                |-> Secret Manager
```

There is no production environment in this release. A pull request runs validation only. A successful push to `main` may deploy staging after the GitHub `staging` Environment rules are satisfied.

## Staging architecture

```text
Internet
   |
   v
rentstage-web-staging (Cloud Run, public)
   |
   | Google-signed ID token in X-Serverless-Authorization
   v
rentstage-api-staging (Cloud Run, private; web runtime SA is run.invoker)
   |
   | /cloudsql/PROJECT:REGION:INSTANCE
   v
Cloud SQL PostgreSQL 18

Identity Platform  <-> browser email/password authentication
Secret Manager     -> DATABASE_URL and request-fingerprint salt
Artifact Registry  <- immutable SHA-tagged API/web images
```

The API is not anonymously invokable. Browsers use the same-origin Next.js proxy at `/api/backend/*`; only the web runtime service account receives `roles/run.invoker` on the API service.

## GitHub workflows

### `.github/workflows/pipeline.yml`

Runs on pull requests, pushes to `main`, and manual dispatches.

| Job | Checks |
|---|---|
| Repository validation | versions, migrations, YAML, Compose, sensitive-file policy, shellcheck, PowerShell parsing |
| API unit | `gofmt`, `go mod verify`, `go test -race`, coverage, `go vet` |
| Web unit | TypeScript, Node unit tests/coverage, optimized Next.js build |
| Terraform validation | `terraform fmt`, initialization without backend, `terraform validate` |
| Source security | Gitleaks, govulncheck, gosec, npm audit, Trivy filesystem/secrets/IaC, optional Dependency Review |
| Integration | clean Docker Compose build, all migrations, helper tests, six product smoke suites |
| Container security | build API/web images and fail on HIGH/CRITICAL findings according to the configured Trivy policy |
| Deploy staging | WIF authentication, image push, image scan, Cloud Run deployment, full/read-only staging smoke, rollback |

### `.github/workflows/infra-staging.yml`

Manual `plan` or `apply` from `main`. The plan is intentionally not uploaded as an artifact because Terraform plan files can contain sensitive values. Protect the `staging` Environment with a required reviewer when you want an approval gate.

### `.github/workflows/codeql.yml`

Runs weekly and on pull requests/pushes when the repository is public or repository variable `ENABLE_CODEQL=true` is available for the account plan.

### `.github/dependabot.yml`

Checks Go, npm, GitHub Actions, and Terraform dependencies weekly.

## 1. Create GitHub and Google Cloud resources

Create an empty GitHub repository and push the current RentStage tree to `main`.

Create a billing-enabled Google Cloud project dedicated to staging. Do not reuse a production project.

In Cloud Shell, authenticate GitHub CLI and clone the repository:

```bash
gh auth login
git clone https://github.com/OWNER/REPOSITORY.git
cd REPOSITORY
```

Run the bootstrap helper as a project owner or bootstrap administrator:

```bash
export PROJECT_ID="your-rentstage-staging-project"
export GITHUB_REPOSITORY="OWNER/REPOSITORY"
export REGION="us-east1"

bash scripts/bootstrap-gcp-staging.sh
```

It creates:

- a private, versioned GCS Terraform-state bucket;
- separate GitHub infrastructure and deployment service accounts;
- a Workload Identity Pool/provider;
- WIF bindings restricted by immutable GitHub repository ID, owner ID, `refs/heads/main`, and Environment `staging`;
- bootstrap IAM needed for Terraform.

No service-account key is generated.

## 2. Configure the GitHub `staging` Environment

Create **Settings -> Environments -> staging**. Restrict deployments to `main`. A required reviewer is recommended before `apply` and application deployment.

Add the Environment variables printed by the bootstrap script:

```text
GCP_PROJECT_ID
GCP_REGION
GCP_WIF_PROVIDER
GCP_INFRA_SERVICE_ACCOUNT
GCP_DEPLOY_SERVICE_ACCOUNT
GCP_ARTIFACT_REPOSITORY
TF_STATE_BUCKET
STAGING_SEED_DEMO_DATA
RUN_FULL_STAGING_SMOKE
```

Recommended initial values:

```text
GCP_REGION=us-east1
GCP_ARTIFACT_REPOSITORY=rentstage
STAGING_SEED_DEMO_DATA=true
RUN_FULL_STAGING_SMOKE=true
```

Add Environment secrets:

```text
STAGING_DATABASE_PASSWORD  24-64 random alphanumeric characters
STAGING_SMOKE_EMAIL         owner@rentstage.local
STAGING_SMOKE_PASSWORD      at least 12 characters
```

`STAGING_SMOKE_EMAIL` must remain `owner@rentstage.local` while demo seeding is enabled because the seed grants that deterministic account the OWNER membership in `AudioPro Demo`.

Optional repository variables:

```text
ENABLE_CODEQL=false
ENABLE_DEPENDENCY_REVIEW=false
```

Enable them only when those GitHub security features are available for the repository/account.

## 3. Create staging infrastructure

Run **Actions -> Staging Infrastructure -> Run workflow** with `operation=plan`. Inspect the Terraform output in the job log.

Then run the same workflow with `operation=apply`. It creates:

- Artifact Registry repository;
- Cloud SQL PostgreSQL 18 instance/database/user, backups, and point-in-time recovery;
- Firebase project/web application and Identity Platform email/password sign-in;
- API and web runtime service accounts;
- Secret Manager values;
- deployment and runtime IAM.

Cloud SQL uses PostgreSQL 18 with the `ENTERPRISE` edition and the low-cost shared-core `db-f1-micro` tier. This is a shared-core staging database for functional validation, not a production SLA or load-test shape. Increase the tier before performance testing.

The database URL and database password are sensitive values in Terraform state. Keep the state bucket restricted and never upload a `tfplan` or state file to GitHub artifacts.

## 4. Deploy staging

Push a new commit to `main` or manually run **RentStage CI/CD**. Deployment begins only after every required job passes.

The deployment workflow:

1. authenticates to Google Cloud through WIF;
2. reads the staging foundation;
3. builds API and web images tagged with the commit SHA;
4. attaches provenance and an SBOM;
5. pushes to Artifact Registry;
6. scans the pushed images;
7. deploys the private API and public web Cloud Run services;
8. grants only the web runtime identity API invocation;
9. registers the generated web hostname in Identity Platform;
10. runs staging smoke tests through the public web proxy;
11. restores the prior revisions automatically if validation fails.

## Deployment serialization

Staging infrastructure changes and application deployments share the same GitHub concurrency group, `rentstage-staging-change`. Terraform apply and Cloud Run deployment therefore cannot mutate staging concurrently.

## Staging smoke modes

`RUN_FULL_STAGING_SMOKE=true` runs:

```text
authentication
packages
public catalog submission
quote portal acceptance
billing and payments
DTE MOCK / TEST
```

The suites create temporary records and use supported cancellation/void/invalidation flows to clean operational state while preserving audit history and consumed sequence numbers.

Set `RUN_FULL_STAGING_SMOKE=false` for read-only checks during an incident or when demo data is disabled. Read-only mode validates health, API privacy, response protections, authentication, packages, and catalog availability.

## Branch protection

Require pull requests and the following checks before merging to `main`:

```text
Repository validation
API unit, race, and vet
Web unit, typecheck, and production build
Terraform format and validation
Source and dependency security
Docker integration and complete smoke suite
Container vulnerability scan
```

Also enable secret scanning/push protection when available, dismiss stale approvals, and block force pushes to `main`.

## Local development remains unchanged

```powershell
Copy-Item .env.example .env
docker compose up -d --build
powershell -ExecutionPolicy Bypass -File .\scripts\run-smoke-suite.ps1
```

Local development continues to use the Firebase emulator and PostgreSQL Docker volume. Never copy the staging database secret or WIF configuration into `.env`.

## Rollback

A failed deployment automatically sends 100% traffic back to the previous API and web revisions. If there was no previous revision, the newly created service is removed.

Manual rollback remains available:

```bash
export GCP_PROJECT_ID="..."
export GCP_REGION="us-east1"
export PREVIOUS_API_REVISION="rentstage-api-staging-..."
export PREVIOUS_WEB_REVISION="rentstage-web-staging-..."
bash scripts/gcp/rollback-staging.sh
```

Database migrations are forward-only. Before introducing a future schema migration, verify that the previous application revision remains compatible or design an explicit expand/contract rollout.
