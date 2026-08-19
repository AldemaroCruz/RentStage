# RentStage v0.16.0 validation record

## Repository and automated contracts

- [ ] `VERSION`, `apps/web/package.json`, README, dashboard link, and changelog report `0.16.0`.
- [ ] Repository version, workflow YAML, migration ordering, sensitive-file, shell, and PowerShell syntax checks pass.
- [ ] Go formatting, module integrity, unit/race tests, and vet pass.
- [ ] Frontend typecheck, 95/90/95 coverage thresholds, and optimized production build pass.
- [ ] Docker integration, vulnerability, security, CodeQL, Terraform, deployment, and staging smoke jobs remain green.

## API and tenant boundaries

- [ ] `GET /api/v1/metrics/commercial` defaults to 30 days.
- [ ] `days=7`, `days=30`, and `days=90` succeed; other values return a validation response.
- [ ] The response contains five funnel stages, six calendar months, four customer sources, reservation outcomes, commercial values, and evidence counts.
- [ ] A user without `operations.read` is forbidden.
- [ ] Switching workspaces changes the report and does not disclose data from the previous tenant.

## Metric behavior

- [ ] Pipeline and outstanding values are clearly labeled as current snapshots.
- [ ] Quote-to-reservation conversion counts only reservations linked to a quote.
- [ ] First-response time exposes its sample size and shows **Sin muestra** when no sent response exists.
- [ ] Manual reservations, undecided quotes, voided invoices, and voided payments do not enter incompatible denominators.
- [ ] The funnel discloses that it is windowed activity rather than a closed cohort.

## Presentation

- [ ] Desktop and mobile layouts preserve readable cards, charts, labels, and window controls.
- [ ] Light and dark themes use shared surface and semantic tokens with no harsh white panel.
- [ ] Zero-value charts and empty source groups remain legible.
- [ ] The DEMO assistant boundary and DTE MOCK / TEST boundary remain explicit.

## Evidence

```text
CI/CD run: ______________________________
Frontend coverage: ______________________
7/30/90 screenshots: ____________________
Tenant-isolation check: _________________
Staging smoke result: ___________________
Staging URL: ____________________________
Commit SHA: _____________________________
Validated by: ___________________________
Date: __________________________________
```
