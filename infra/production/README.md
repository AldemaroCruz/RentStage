# RentStage production infrastructure

This Terraform root calls the same platform module as staging but uses an independent Google Cloud project, state bucket, GitHub Environment, Workload Identity Federation pool, and service accounts.

Version 0.17.0 intentionally provides a **plan-only** production workflow. It does not deploy Cloud Run, populate Meta credentials, seed demo data, or expose a production sender. The first apply must be introduced only after cost, IAM, database availability, domains, and the complete plan have been reviewed.

The bootstrap gives the planning identity project Viewer, Service Usage Consumer, and object administration only on its isolated state bucket. It also creates a reserved deployment service account, but does not let GitHub impersonate it. The Terraform root deliberately sets `enable_deploy_iam_bindings = false`, so production receives no deployment project roles or runtime impersonation until a later approved deploy release.

Production defaults to a single-zone `db-custom-1-3840` Cloud SQL instance to avoid silently enabling a high fixed cost before RentStage has customers. Change `database_availability_type` to `REGIONAL` only after reviewing its price and recovery objective. Backups, PITR, TLS, connector enforcement, and deletion protection remain enabled.

Terraform creates empty Secret Manager containers for Meta identifiers and credentials. Add secret versions out of band after Meta Business verification so no token or application secret enters a Terraform plan or state file.

The Cloud SQL password and generated database URL are present in Terraform state. The production state bucket must be private, versioned, protected from public access, and accessible only to the production infrastructure identity.

## Required review order

1. Create a new billing-enabled production project and a budget alert.
2. Run `scripts/bootstrap-gcp-production.sh` from Cloud Shell.
3. Create and protect the GitHub `production` Environment using the emitted values.
4. Run **Production Infrastructure Plan** from `main`.
5. Review every planned resource and the Cloud SQL monthly estimate.
6. Do not apply until an explicit production-apply increment adds a second approval and documented rollback/recovery procedure.
