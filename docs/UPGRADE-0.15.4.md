# Upgrade RentStage v0.15.3 → v0.15.4

v0.15.4 completes the internal dark-theme surface contract introduced in v0.15.3. It is a presentation-quality frontend patch and does not change business behavior.

## What changes

- Dashboard metrics and the guided-demo strip use theme surfaces instead of fixed white backgrounds.
- Calendar controls, weekdays, days, loading overlays, assignments, and attention summaries preserve readable dark contrast.
- Package grids, quote and reservation metrics, table headers, web requests, customers, warehouse controls, and dialogs share the same dark tokens.
- Search boxes and prefixed fields render as one aligned control because their inner input remains transparent.
- Previously referenced purple, muted-ink, and soft-shadow aliases are defined from the canonical design tokens.
- Printing a dark browser session temporarily restores light document tokens.

## Compatibility

This release changes no API endpoint, database migration, Terraform resource, GitHub workflow, environment variable, secret, IAM permission, role, tenant boundary, assistant action, quote lifecycle, reservation rule, or public bearer-token contract.

## Deployment

Deploy through the existing `pipeline.yml` while staging is resumed and `STAGING_DEPLOY_ENABLED=true`. No Terraform apply or database operation is required.

After deployment, switch between light and dark mode and inspect Dashboard, Calendario, Paquetes, Cotizaciones, Solicitudes web, Reservas, Clientes, and WhatsApp AI at desktop and mobile widths.

## Rollback

Rolling back the application restores the v0.15.3 presentation styles. No data or infrastructure rollback is necessary.
