# RentStage v0.13.1 build and smoke-suite hotfix

## Symptoms

- API image build fails during `go test ./...` with missing `go.sum` entries
  for `firebase.google.com/go/v4` and `firebase.google.com/go/v4/auth`.
- Windows reports that `pwsh` is not recognized when the aggregate smoke suite
  is launched from Windows PowerShell 5.1.

## Cause

Earlier Docker builds ran `go mod tidy` only inside an image layer. The complete
Firebase dependency graph therefore did not reach the repository's `go.sum`.
v0.13.0 moved unit tests before that implicit tidy step and correctly exposed
this stale module metadata.

The aggregate smoke wrapper also assumed PowerShell 7 was installed locally.
GitHub-hosted runners provide `pwsh`, but a default Windows installation may
only expose `powershell.exe`.

## Resolution

1. Apply the v0.13.1 overlay.
2. Run `scripts/ci/sync-go-modules.ps1` once and commit the resulting
   `apps/api/go.mod` and `apps/api/go.sum`.
3. Rebuild API and web.
4. Run `scripts/ci/run-smoke-suite.ps1` with Windows PowerShell or PowerShell 7.

No migration or business-data change is included.
