# RentStage v0.13.2 — Go module synchronization shell fix

## Symptom

On Windows Docker Desktop, the v0.13.1 helper could fail after pulling the
correct image:

```text
sh: go: not found
Go module synchronization failed with exit code 127.
```

The image is valid. The helper started Alpine with `sh -lc`. The `-l` option
creates a login shell, which reads `/etc/profile` and resets `PATH`; that removes
`/usr/local/go/bin`, which the official Go image adds through its image
environment.

## Fix

v0.13.2 does not start a shell. It runs the Go binary directly:

```text
Docker --entrypoint /usr/local/go/bin/go
```

The helper now runs three explicit commands:

```text
go version
go mod tidy
go mod verify
```

This avoids differences between Alpine login shells, Windows PowerShell
argument handling, and the container image PATH.

## Apply

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

Expand-Archive `
  -Path "$HOME\Downloads\rentstage-upgrade-v0.13.2.zip" `
  -DestinationPath . `
  -Force

Get-Content .\VERSION
```

Expected:

```text
0.13.2
```

The application containers can remain stopped while the module files are
synchronized.

## Synchronize module metadata

```powershell
powershell `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\sync-go-modules.ps1
```

Confirm Firebase checksums:

```powershell
Select-String `
  -Path .\apps\api\go.sum `
  -Pattern "firebase.google.com/go/v4"
```

Then rebuild and run the smoke suite:

```powershell
docker compose build --no-cache api web
if ($LASTEXITCODE -ne 0) { throw "Docker build failed." }

docker compose up -d

powershell `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\ci\run-smoke-suite.ps1
```

Commit both module files before opening the pull request:

```powershell
git add apps/api/go.mod apps/api/go.sum
git commit -m "fix: synchronize Go module metadata"
```

## Compatibility

- No database migration.
- No `.env` change.
- No business-data change.
- No Firebase, DTE, billing, inventory, or public-catalog behavior change.
