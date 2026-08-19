# Upgrade RentStage v0.15.4 → v0.15.5

v0.15.5 completes the administrative dark-theme pass begun in v0.15.3 and expanded in v0.15.4. It is a frontend presentation patch and does not change application behavior.

## What changes

- DTE list and integration banners use semantic dark gradients for MOCK and MH_HTTP modes.
- Public-catalog package and resource editors use themed cards, controls, borders, and readable secondary copy.
- Quote Portal prominent controls and its SHA-256 security notice no longer introduce fixed white panels.
- Billing's sticky save action follows the active surface tokens.
- Audit timeline events and markers use the same dark surface hierarchy as the rest of the application.
- Focused stylesheet-contract tests prevent those component families from regressing.

## Compatibility

This release changes no API endpoint, database migration, Terraform resource, GitHub workflow, environment variable, secret, IAM permission, role, tenant boundary, assistant action, quote lifecycle, billing calculation, DTE transition, audit record, or public bearer-token contract.

## Deployment

Deploy through the existing `pipeline.yml` while staging is resumed and `STAGING_DEPLOY_ENABLED=true`. No Terraform apply or database operation is required.

After deployment, switch to dark mode and inspect DTE, Catálogo público, Portal de cotización, Facturación, Integración DTE, and Auditoría. Then return to light mode and verify those screens remain unchanged.

## Rollback

Rolling back the application restores the v0.15.4 presentation styles. No data or infrastructure rollback is necessary.
