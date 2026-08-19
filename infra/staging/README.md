# RentStage staging infrastructure

This Terraform root calls the shared `infra/modules/rentstage-platform` module with staging-safe values. Cloud Run revisions are deployed by `.github/workflows/pipeline.yml`, not by Terraform.

Terraform manages:

- Artifact Registry;
- PostgreSQL 18 Cloud SQL instance using the ENTERPRISE edition and low-cost shared-core `db-f1-micro` tier, plus database, user, backups, and PITR;
- API and web runtime service accounts;
- Firebase project, web app, API-restricted web key, and Identity Platform email/password sign-in;
- Secret Manager entries for the database URL, public-request fingerprint salt, and Firebase API key;
- least-purpose runtime and deployment IAM bindings.

The generated Cloud Run web hostname is unknown during the first Terraform apply. The deployment workflow adds that exact hostname to Identity Platform after creating the web service. Terraform intentionally ignores later changes to `authorized_domains` so it does not remove the dynamic hostname.

The database password and generated database URL are present in Terraform state. Keep the GCS state bucket private, versioned, and restricted to the infrastructure identity. Binary Terraform plans can also contain sensitive values and are intentionally not uploaded as GitHub artifacts.

Run the `Staging Infrastructure` workflow first with `plan` and inspect its log. Then run it with `apply` from `main`; protect the GitHub `staging` Environment with a reviewer for an approval gate.

Version 0.17.0 moves the existing Terraform addresses under `module.platform`. The declarations in `moved.tf` preserve the current remote resources. The first plan after upgrading must report address moves and **zero resources to add, replace, or destroy**. Do not apply if it proposes infrastructure recreation.

The default shared-core tier is intentionally staging-only and should be increased before load or production testing. Both Terraform-level and Google Cloud-level deletion protection follow `var.deletion_protection`.
