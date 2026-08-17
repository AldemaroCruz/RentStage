#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
API_DIR="$ROOT/apps/api"
IMAGE="golang:1.26.6-alpine"

command -v docker >/dev/null 2>&1 || {
  echo "Docker is required to synchronize the Go module graph." >&2
  exit 1
}

args=(
  --rm
  --mount "type=bind,source=$API_DIR,target=/src"
  --workdir /src
  --entrypoint /usr/local/go/bin/go
)

if [[ "$(uname -s)" == "Linux" ]]; then
  args+=(--user "$(id -u):$(id -g)")
fi

run_go() {
  local description="$1"
  shift
  printf '==> %s\n' "$description"
  docker run "${args[@]}" "$IMAGE" "$@"
}

run_go "Checking the pinned Go toolchain" version
run_go "Regenerating apps/api/go.mod and apps/api/go.sum" mod tidy
run_go "Verifying the synchronized Go module graph" mod verify

grep -Eq '^firebase\.google\.com/go/v4 v4\.21\.0 h1:' "$API_DIR/go.sum" || {
  echo "go.sum still does not contain the Firebase Admin SDK v4.21.0 checksum." >&2
  exit 1
}

printf '\n%s\n' \
  'Go module metadata is complete and verified.' \
  'Commit both apps/api/go.mod and apps/api/go.sum before opening the pull request.'

git -C "$ROOT" diff -- apps/api/go.mod apps/api/go.sum || true
