# RentStage v0.13.0 validation record

## Release contracts

- Incremental update only from v0.12.1.
- No new SQL migration.
- Local Docker Compose remains compatible with the existing data volumes.
- Staging is the only remote environment.
- DTE remains MOCK / TEST in automated staging validation.

## Automated CI coverage

The included workflows execute:

- repository version, migration, YAML, Compose, sensitive-file, shell, and PowerShell checks;
- Go formatting, module verification, unit tests, race detector, coverage, and vet;
- Node unit tests and coverage, TypeScript, and Next.js production build;
- Terraform format and provider-schema validation;
- Gitleaks, govulncheck, gosec, npm audit, Trivy source/IaC/image scans;
- clean Docker Compose build and all product smoke tests;
- optional Dependency Review and CodeQL where the GitHub plan supports them;
- staging deployment and public/private boundary validation.

## Security invariants

- GitHub uses OIDC Workload Identity Federation, not a downloadable service-account key.
- The federation provider is restricted to immutable repository/owner IDs, `main`, and the `staging` Environment.
- API Cloud Run has no `allUsers` invoker binding.
- Web-to-API calls carry a metadata-server identity token in `X-Serverless-Authorization`.
- Staging cookies are secure and application URLs must be HTTPS origins.
- Firebase emulator and local bootstrap are rejected outside local development.
- Cloud SQL credentials and anti-abuse salt are injected from Secret Manager.
- Terraform state and plan files are excluded from source and artifacts.
- The generated Cloud Run hostname is registered in Identity Platform without disabling Terraform ownership of the base authentication configuration.
- Images are SHA-tagged, scanned, and published with provenance/SBOM metadata.

## Runtime acceptance still required

The first real GCP execution must validate:

1. bootstrap permissions and WIF propagation;
2. Terraform plan/apply in the selected billing-enabled project;
3. Identity Platform project creation and email/password sign-in;
4. Cloud SQL PostgreSQL 18 availability in the selected region/tier;
5. private API invocation from the web runtime identity;
6. Cloud Run hostname authorization in Identity Platform;
7. full staging smoke suite;
8. rollback to a previous revision after an intentionally failed validation.

No claim is made that GCP resources were provisioned from this packaging environment. The included source can be statically and locally validated; cloud acceptance occurs in the user's staging project.
