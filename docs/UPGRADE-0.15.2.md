# Upgrade RentStage v0.15.1 → v0.15.2

v0.15.2 is a test-quality increment. It expands coverage for frontend business-formatting and Cloud Run authentication helpers and turns coverage into a blocking CI contract.

## Coverage contract

`npm run test:coverage` now fails when the measured frontend library coverage falls below:

- 95% executable lines;
- 90% branches;
- 95% functions.

The same command remains part of the existing `test:ci` chain before the production Next.js build. No workflow, job topology, staging flag, or deployment dependency is added.

## Deployment

Deploy through the existing `pipeline.yml`. This increment contains test, documentation, and displayed-version changes only. It requires no database migration, Terraform operation, Cloud SQL resume, secret, IAM grant, or service restart beyond the normal application deployment.

## Rollback

Application rollback removes only the stronger test suite, thresholds, and displayed v0.15.2 label. Runtime data and the v0.15.1 WhatsApp demo remain compatible.
