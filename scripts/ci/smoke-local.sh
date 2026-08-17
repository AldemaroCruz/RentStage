#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"
mkdir -p artifacts
cleanup() {
  status=$?
  docker compose ps -a >artifacts/compose-ps.txt 2>&1 || true
  docker compose logs --no-color >artifacts/compose.log 2>&1 || true
  docker compose down -v --remove-orphans >/dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

docker compose down -v --remove-orphans || true
docker compose build --no-cache
docker compose up -d
bash scripts/ci/wait-local.sh
pwsh -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/test-smoke-common.ps1
pwsh -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/run-smoke-suite.ps1
