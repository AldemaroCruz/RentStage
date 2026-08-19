# RentStage v0.17.0 validation record

## Repository contracts

- [ ] `VERSION`, `apps/web/package.json`, README, dashboard release label, and changelog report `0.17.0`.
- [ ] Version, YAML, migration-ordering, sensitive-file, environment-isolation, shell, PowerShell, and `git diff --check` contracts pass.
- [ ] Both `infra/staging` and `infra/production` pass `terraform fmt -check`, backend-disabled initialization, and `terraform validate`.
- [ ] Existing API, frontend, Docker, CodeQL, dependency, Trivy, and staging smoke jobs remain green.

## Staging preservation

- [ ] The remote staging plan uses `rentstage/staging` state.
- [ ] Existing resources move under `module.platform`.
- [ ] The staging plan reports zero additions, replacements, and destructions.
- [ ] Cloud SQL remains `rentstage-staging-postgres`, ZONAL, deletion-protected, TLS-only through the connector, with backups and PITR.
- [ ] Existing Firebase, runtime service accounts, secret IDs, Artifact Registry, and application deploy behavior are unchanged.

## Production isolation

- [ ] The production project differs from the staging project.
- [ ] The `production` GitHub Environment is limited to `main` and has a required reviewer.
- [ ] Production uses its own WIF provider, infrastructure/deployment identities, state bucket, and `rentstage/production` prefix.
- [ ] `PRODUCTION_DATABASE_PASSWORD` is unique and is not present in source, logs, or a Terraform variables file.
- [ ] The workflow plan targets only the production project and contains no `terraform apply` or Cloud Run deploy step.
- [ ] The plan creates production-suffixed runtime accounts, database, and secrets without referencing staging names.
- [ ] The production deployment service account receives no project IAM role, runtime impersonation, or Firebase-key access in this release.
- [ ] The production planning identity is project read-only and can mutate only its isolated Terraform state objects.
- [ ] GitHub cannot impersonate the reserved production deployment identity in this release.
- [ ] Database cost, ZONAL/REGIONAL choice, region, backups, deletion protection, and recovery objective have named reviewers.

## Meta boundary

- [ ] Terraform plans six empty Meta Secret Manager containers and no secret versions.
- [ ] No Meta token, app secret, webhook token, WABA ID, or phone-number ID appears in Terraform state, GitHub variables, build arguments, source, or logs.
- [ ] The application remains in `DEMO`; no real phone receives a message.
- [ ] The next provider increment requires signed/idempotent webhooks, human-approved delivery, consent/service-window/template enforcement, status ingestion, observability, and an emergency off switch.

## Evidence

```text
CI/CD run: ______________________________
Staging plan run: _______________________
Staging add/change/destroy: _____________
Production plan run: ____________________
Production project ID: __________________
Production state bucket/prefix: _________
Cloud SQL estimate reviewed by: _________
IAM review: _____________________________
Commit SHA: _____________________________
Validated by: ___________________________
Date: __________________________________
```
