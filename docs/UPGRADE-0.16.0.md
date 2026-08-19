# Upgrade RentStage v0.15.5 → v0.16.0

v0.16.0 adds operational metrics over existing RentStage records. It introduces one authenticated read endpoint and one admin route, but no schema migration or infrastructure change.

## What changes

- New API module: `internal/core/commercialmetrics`.
- New endpoint: `GET /api/v1/metrics/commercial?days=7|30|90`.
- New admin route and navigation item: `/metrics`.
- The dashboard release shortcut opens the metrics workspace.
- The authenticated smoke flow validates the report shape.

## Compatibility

The release reuses `operations.read`; it adds no role, permission, migration, Terraform resource, service account, secret, environment variable, build argument, or external analytics dependency. Existing sessions and tenant data remain valid.

## Deployment

1. Resume staging Cloud SQL if it is paused.
2. Set `STAGING_DEPLOY_ENABLED=true`.
3. Apply the v0.16.0 increment, run the repository checks, commit, and push to `main`.
4. Let the existing single RentStage CI/CD workflow build, scan, deploy, and execute staging smoke tests.
5. Sign in and open **Métricas**. Exercise 7, 30, and 90-day windows and confirm that the workspace identifies current snapshots separately from windowed activity.

Terraform plan/apply and a manual database migration are not required.

## Rollback

Rolling the API and web services back to v0.15.5 removes the metrics route and endpoint. Because v0.16.0 writes no data and changes no schema, no database or infrastructure rollback is required.
