# RentStage production infrastructure foundation v0.17.0

RentStage now uses one reviewed Terraform module with two independent roots:

| Boundary | Staging | Production |
| --- | --- | --- |
| Terraform root | `infra/staging` | `infra/production` |
| GCP project | Existing staging project | New production-only project |
| GitHub Environment | `staging` | `production` |
| State prefix | `rentstage/staging` | `rentstage/production` |
| WIF provider | Staging-bound | Production-bound |
| Runtime accounts | `rentstage-api-stg`, `rentstage-web-stg` | `rentstage-api-prod`, `rentstage-web-prod` |
| Database | `rentstage-staging-postgres` | `rentstage-production-postgres` |
| Meta secret containers | Disabled | Empty containers only |
| Workflow mutation | Existing reviewed staging apply | Plan only |

Changing only `project_id` in the staging root would be unsafe because its state, GitHub Environment, service accounts, approval policy, resource names, and operational lifecycle would remain coupled to staging. The separate roots keep these concerns isolated while the shared module prevents configuration drift.

## What the production plan contains

- required Google APIs;
- Artifact Registry;
- production API and web runtime identities;
- PostgreSQL 18 Cloud SQL, database, user, backups, PITR, TLS, connector enforcement, and deletion protection;
- Firebase project, web app, restricted browser key, and Identity Platform email/password sign-in;
- database, fingerprint, Firebase, and future Meta Secret Manager containers;
- purpose-specific runtime IAM bindings.

The production planning identity receives project Viewer plus Service Usage Consumer and object administration only on its isolated state bucket. The reserved deployment identity is created for future review but receives no project roles, runtime impersonation, or WIF impersonation binding in this release. The separately reviewed apply/deploy increment must introduce its own least-privilege grants. Staging preserves its existing deployment permissions because its workflow already creates Cloud Run services and registers the generated web hostname in Identity Platform.

The default production database is `db-custom-1-3840`, `ZONAL`, 20 GiB SSD, with 14 retained backups. `REGIONAL` availability is available but is not silently enabled because it materially increases fixed cost. Review the current Google Cloud estimate and the required recovery objective before changing it.

## What v0.17.0 cannot do

- apply the production plan;
- create Cloud Run services;
- publish a production image;
- seed demo tenants or users;
- populate Meta credentials;
- register a WhatsApp sender or webhook;
- enable `PRODUCTION_DEPLOY_ENABLED`;
- destroy or recreate staging.

## State and secret rules

The production state bucket is separate, private, versioned, and protected from public access. The production database password and the derived database URL are sensitive but necessarily represented in Terraform state, so state access is production-infrastructure access.

Meta credential **containers** are declared in Terraform, but their versions must be inserted out of band. Never place an access token, app secret, webhook verification token, private key, downloaded credential JSON, or binary plan in Git.

## Safe adoption order

1. Create a dedicated GCP project with billing and a budget alert.
2. Authenticate `gcloud` and `gh` in Cloud Shell.
3. Run `scripts/bootstrap-gcp-production.sh` with the new project and repository.
4. Create a protected GitHub `production` Environment and enter the emitted variables and database secret.
5. Run **Production Infrastructure Plan** from `main`.
6. Review every addition, Cloud SQL cost, IAM binding, region, domain, and deletion guard.
7. Separately run the staging infrastructure plan and confirm only state-address moves, with zero create/replace/destroy actions.
8. Stop. A later release must add an approved production apply and recovery procedure.

## Rollback

Before any production apply, rollback is code-only: revert v0.17.0. The production bootstrap may have created two service accounts, a WIF pool/provider, and an empty state bucket; retain them until the plan review is closed, then remove them manually only if the production project is abandoned.

The staging `moved` declarations should remain after adoption. Removing them before every workspace has migrated can make Terraform interpret existing resources as new addresses.
