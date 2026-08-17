#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT/apps/api"

mkdir -p "$ROOT/artifacts/coverage" "$ROOT/artifacts/security"

# Keep dependency metadata reproducible in Git. This also produces a clear CI
# error when go.mod/go.sum were previously generated only inside Docker.
go mod tidy
if ! git diff --exit-code -- go.mod go.sum; then
  echo "::error::Go module metadata is not tidy. Run scripts/ci/sync-go-modules.ps1, then commit apps/api/go.mod and apps/api/go.sum."
  exit 1
fi

go mod download
go mod verify
go test -race -shuffle=on -covermode=atomic \
  -coverprofile="$ROOT/artifacts/coverage/go-cover.out" ./...
go vet ./...

echo "Backend unit tests, race detector, coverage and vet passed."
