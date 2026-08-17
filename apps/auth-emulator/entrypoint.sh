#!/bin/sh
set -eu

PROJECT_ID="${FIREBASE_PROJECT_ID:-demo-rentstage}"
mkdir -p /data

IMPORT_ARGS=""
if [ -f /data/firebase-export-metadata.json ]; then
  IMPORT_ARGS="--import=/data"
fi

exec firebase emulators:start \
  --project "$PROJECT_ID" \
  --only auth \
  --config /app/firebase.json \
  $IMPORT_ARGS \
  --export-on-exit=/data
