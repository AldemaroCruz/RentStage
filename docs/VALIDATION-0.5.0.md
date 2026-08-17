# RentStage v0.5.0 validation record

## Completed in the build environment

- `gofmt` over every Go source file.
- Static compilation of every Go package against a local contract-compatible `pgx`/`pgxpool` harness.
- Go unit tests, including new manual-reservation, schedule-boundary, reschedule, and status-filter tests.
- `go vet` against the same harness.
- Strict TypeScript validation against local React/Next framework contracts.
- JSON parsing for frontend configuration.
- Docker Compose YAML parsing.
- CSS brace-balance validation.
- Internal route/import review.
- Version and migration-order checks.
- Full-package and upgrade-package integrity checks.
- Upgrade-overlay comparison against the complete v0.5 project.

## Domain cases covered by code/tests

- Server-side manual-reservation totals.
- Event period must be inside the blocked period.
- Duplicate resource rejection.
- Availability check excludes the reservation being rescheduled.
- Rescheduling is limited to pre-check-out states.
- Exact assigned-asset conflict is checked separately from quantity capacity.
- Source and quote relationship is enforced by PostgreSQL constraints.
- Schedule changes write immutable history.
- Operations status query rejects unsupported values.

## Runtime checks still required on the user's computer

This environment does not provide a Docker daemon, PostgreSQL server, or external package resolution. The definitive integration test is therefore:

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a
```

Then verify:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

And confirm `005_calendar_operations.sql` in `schema_migrations`.

## Recommended acceptance scenario

1. Create a manual reservation with multiple resources.
2. Verify it blocks overlapping capacity.
3. Find it in month/week/agenda views.
4. Reprogram it to a free period.
5. Confirm schedule history and audit.
6. Create a second reservation that consumes remaining capacity.
7. Attempt to move the first reservation into conflict and verify HTTP 409.
8. Assign a physical asset, then test a new date where that exact unit is committed elsewhere.
9. Verify the structured physical-unit conflict.
10. Advance a reservation into an overdue state and confirm the alert panel and bell count.
