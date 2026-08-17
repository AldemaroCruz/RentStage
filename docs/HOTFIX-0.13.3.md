# RentStage v0.13.3 — Frontend TypeScript test-import compatibility

## Symptom

The v0.13.2 Docker build reaches the frontend CI contract and stops during
`tsc --noEmit` with `TS5097` for these tests:

```text
lib/cloud-run-auth.test.ts
lib/format.test.ts
lib/navigation.test.ts
lib/proxy-network.test.ts
```

The tests intentionally import TypeScript source modules using explicit `.ts`
specifiers so Node's native TypeScript test runner can resolve them. The
TypeScript project had `noEmit: true` but had not enabled
`allowImportingTsExtensions`.

## Fix

The web `tsconfig.json` now includes:

```json
{
  "compilerOptions": {
    "allowImportingTsExtensions": true,
    "noEmit": true
  }
}
```

This keeps the explicit test imports and allows the existing `test:ci` chain to
continue through type checking, coverage tests, and the production Next.js
build.

## Apply

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

docker compose down

Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.13.3.zip" `
  -DestinationPath . `
  -Force

Get-Content .\VERSION
```

Expected:

```text
0.13.3
```

Build the web image first:

```powershell
docker compose build --no-cache web
if ($LASTEXITCODE -ne 0) { throw "Web build failed." }
```

Then build the API and start the stack:

```powershell
docker compose build api
if ($LASTEXITCODE -ne 0) { throw "API build failed." }

docker compose up -d
Start-Sleep -Seconds 12
docker compose ps -a
```

Finally run the complete smoke suite:

```powershell
powershell `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\run-smoke-suite.ps1
```

## Compatibility

- No database migration.
- No `.env` change.
- No application runtime behavior change.
- No GCP or GitHub Actions configuration change.
