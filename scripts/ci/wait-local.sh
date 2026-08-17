#!/usr/bin/env bash
set -euo pipefail
for attempt in $(seq 1 90); do
  if curl --fail --silent http://127.0.0.1:8080/readyz >/dev/null \
    && curl --fail --silent http://127.0.0.1:3000/api/healthz >/dev/null; then
    docker compose ps
    exit 0
  fi
  sleep 2
done
docker compose ps -a >&2
docker compose logs --tail=300 >&2 || true
exit 1
