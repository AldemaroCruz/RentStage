SHELL := /bin/bash
POWERSHELL ?= pwsh

.PHONY: up build down restart logs ps config reset api-logs web-logs auth-logs db-logs db-shell \
	smoke smoke-packages smoke-public smoke-quote-portal smoke-billing smoke-dte smoke-all \
	validate test-api test-web security ci-local compose-up compose-down

up:
	docker compose up --build -d
	@echo "RentStage login:       http://localhost:$${WEB_PORT:-3000}/login"
	@echo "RentStage API:         http://localhost:$${API_PORT:-8080}"
	@echo "Public demo catalog:   http://localhost:$${WEB_PORT:-3000}/p/audiopro-demo"
	@echo "Auth Emulator UI:      http://localhost:$${AUTH_EMULATOR_UI_PORT:-4000}"
	@echo "Auth Emulator API:     http://localhost:$${AUTH_EMULATOR_PORT:-9099}"

build:
	docker compose build api auth web

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

api-logs:
	docker compose logs -f api

web-logs:
	docker compose logs -f web

auth-logs:
	docker compose logs -f auth

db-logs:
	docker compose logs -f db

ps:
	docker compose ps -a

config:
	docker compose config

reset:
	docker compose down -v --remove-orphans
	docker compose up --build -d

smoke:
	$(POWERSHELL) -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-auth.ps1

smoke-packages:
	$(POWERSHELL) -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-packages.ps1

smoke-public:
	$(POWERSHELL) -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-public-catalog.ps1

smoke-quote-portal:
	$(POWERSHELL) -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-quote-portal.ps1

smoke-billing:
	$(POWERSHELL) -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-billing.ps1

smoke-dte:
	$(POWERSHELL) -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-dte.ps1

smoke-all:
	$(POWERSHELL) -NoLogo -NoProfile -ExecutionPolicy Bypass -File ./scripts/run-smoke-suite.ps1

db-shell:
	docker compose exec db psql -U $${POSTGRES_USER:-rentstage} -d $${POSTGRES_DB:-rentstage}

validate:
	bash scripts/ci/check-version.sh
	python3 scripts/ci/check-migrations.py
	bash scripts/ci/check-sensitive-files.sh
	python3 scripts/ci/check-workflow-yaml.py
	docker compose config >/dev/null

# Requires Go 1.26.6.
test-api:
	cd apps/api && test -z "$$(gofmt -l .)" && go mod verify && go test -race -shuffle=on -count=1 ./... && go vet ./...

# Requires Node.js 24.
test-web:
	cd apps/web && npm install --no-audit --no-fund && npm run test:ci

security:
	bash scripts/ci/check-sensitive-files.sh
	cd apps/api && govulncheck ./... && gosec -exclude-generated ./...
	cd apps/web && npm audit --audit-level=high

ci-local: validate test-api test-web

compose-up: up

compose-down: down
