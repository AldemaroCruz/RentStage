# RentStage production infrastructure

This Terraform root calls the same platform module as staging but uses an independent Google Cloud project, state bucket, GitHub Environment, Workload Identity Federation pool, and service accounts.

Version 0.17.1 keeps the read-only **Production Infrastructure Plan** workflow and adds a separate, protected **Production Infrastructure Apply** workflow. It does not deploy Cloud Run, populate Meta credentials, seed demo data, or expose a production sender. Apply remains manual, create-only, and disabled by default.

The bootstrap gives the planning identity project Viewer, Service Usage Consumer, and object administration only on its isolated state bucket. It creates a separate apply identity in a different WIF pool restricted to the exact apply workflow; that identity receives state-bucket access but no persistent project mutation role. `scripts/gcp/production-apply-access.sh` grants and revokes its reviewed control-plane roles just in time. The reserved deployment service account still has no project role or WIF binding. The Terraform root deliberately sets `enable_deploy_iam_bindings = false`.

Production API runtime uses a custom Firebase role limited to `firebaseauth.users.get` and `firebaseauth.users.createSession`. Staging retains its existing Firebase Auth Admin binding in this increment so its already-migrated plan remains zero-change.

Production defaults to a single-zone `db-custom-1-3840` Cloud SQL instance to avoid silently enabling a high fixed cost before RentStage has customers. Change `database_availability_type` to `REGIONAL` only after reviewing its price and recovery objective. Backups, PITR, TLS, connector enforcement, and deletion protection remain enabled.

Terraform creates empty Secret Manager containers for Meta identifiers and credentials. Add secret versions out of band after Meta Business verification so no token or application secret enters a Terraform plan or state file.

The Cloud SQL password and generated database URL are present in Terraform state. The production state bucket must be private, versioned, protected from public access, and accessible only to the production infrastructure identity.

## Required review order

1. Create a new billing-enabled production project and a budget alert.
2. Run `scripts/bootstrap-gcp-production.sh` from Cloud Shell.
3. Create and protect the GitHub `production` Environment using the emitted values.
4. Run **Production Infrastructure Plan** from `main`.
5. Review every planned resource and the Cloud SQL monthly estimate.
6. Keep `PRODUCTION_INFRA_APPLY_ENABLED=false` until the v0.17.1 apply checklist is approved.
7. Grant the apply identity's temporary roles, enable the gate, and run **Production Infrastructure Apply** with both confirmations.
8. Disable the gate and revoke the temporary roles immediately after the run, successful or not.
9. Run **Production Infrastructure Plan** again and require `0 to add, 0 to change, 0 to destroy`.

The apply plan file and JSON representation are removed even on failure and are never uploaded as artifacts. See `docs/PRODUCTION-APPLY-0.17.1.md` for exact commands, recovery steps, and stop conditions.
