# RentStage v0.14.0 validation record

## Release contracts

- [ ] `VERSION`, `apps/web/package.json`, README, and changelog report `0.14.0`.
- [ ] Workflow YAML, migrations, sensitive-file policy, PowerShell syntax, and `git diff --check` pass.
- [ ] No new migration, environment variable, GCP resource, or permission is introduced.

## Web checks

Run from `apps/web`:

```bash
npm install --no-audit --no-fund
npm run typecheck
npm run test:coverage
npm run build
```

Expected focused coverage:

- a complete commercial scenario reports five of five ready steps;
- an empty workspace reports zero readiness;
- production DTE is never labeled as the safe demonstration boundary;
- issued revenue without a collected payment does not complete the billing step.

## Seed and integration checks

Start a clean local environment so the embedded migrations and seed execute in order:

```powershell
docker compose down --volumes
docker compose up --build --detach

pwsh `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\test-smoke-common.ps1
```

Confirm the demo tenant contains:

- accepted quote `50000000-0000-0000-0000-000000000003` totaling `$299.00`;
- confirmed reservation `80000000-0000-0000-0000-000000000001` with five items;
- partially paid invoice `INV-140001` totaling `$299.00` with `$150.00` paid;
- payment reference `DEMO-CONAMYPE` allocated to that invoice.

Restart the API once and confirm the stable IDs do not create duplicate records.

## Manual presentation checks

- [ ] `/demo` renders at desktop and mobile widths without horizontal overflow.
- [ ] The owner sees a 100% readiness score on the seeded demo tenant.
- [ ] Journey links open the accepted quote, active reservation, recent invoice, and DTE inbox.
- [ ] The public-catalog action opens `/p/audiopro-demo` in a separate tab.
- [ ] Dashboard **Iniciar demo** opens the guided journey.
- [ ] DTE is labeled `MOCK · TEST`, never as a production transmission.
- [ ] WhatsApp + AI is explicitly labeled as roadmap.
- [ ] A role without an optional finance/fiscal permission does not crash the demo page.

## CI/CD evidence

Record the green GitHub Actions run ID after merge:

```text
RentStage CI/CD run: ____________________
Commit SHA: _____________________________
Staging URL: ____________________________
Validated by: ___________________________
Date: __________________________________
```
