# Upgrade RentStage v0.19.0 → v0.19.1

v0.19.1 hardens the existing first-party web chat without changing its API, database schema, human-approval boundary, infrastructure, or external-provider behavior.

No migration, environment variable, secret, IAM permission, Terraform apply, Meta account, WebSocket, service worker, or telephone number is required.

## 1. Source validation

From the repository root:

```powershell
bash scripts/ci/check-version.sh
python .\scripts\ci\check-workflow-yaml.py
python .\scripts\ci\check-migrations.py
bash scripts/ci/check-sensitive-files.sh

docker run --rm `
  -v "${PWD}:/workspace" `
  -w /workspace/apps/api `
  golang:1.26.6-alpine `
  sh -c 'test -z "$(gofmt -l .)" && go test ./...'

docker run --rm `
  -v "${PWD}:/workspace" `
  -v "rentstage-web-v0191-modules:/workspace/apps/web/node_modules" `
  -w /workspace/apps/web `
  node:24-alpine `
  sh -c "npm install --package-lock=false --no-audit --no-fund && npm run test:ci"

git diff --check
git status --short