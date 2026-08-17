# RentStage v0.12.0 — DTE smoke-test hotfix

This hotfix replaces only `scripts/smoke-dte.ps1`.

It does not modify the API, web application, PostgreSQL schema, `.env`, or persistent volumes.

## Changes

- Uses the deterministic AudioPro demo customer instead of whichever customer was updated most recently.
- Applies known-valid temporary issuer and receiver fiscal data.
- Identifies each profile update separately in the console.
- Surfaces API validation details, including `error`, `message`, `fields`, and `request_id`, when a request returns HTTP 422.
- Restores the original customer, billing, and DTE settings after the test on a best-effort basis.

## Apply

From the RentStage project root in PowerShell:

```powershell
Expand-Archive `
  -Path "$HOME\Downloads\rentstage-hotfix-v0.12.0-smoke-dte.zip" `
  -DestinationPath . `
  -Force
```

No Docker rebuild or restart is required.

Run:

```powershell
powershell `
  -ExecutionPolicy Bypass `
  -File .\scripts\smoke-dte.ps1
```
