# Validation record — RentStage v0.3.0

## Completed before packaging

### Backend

- Go formatting applied to all new and modified `.go` files.
- All RentStage Go packages compiled against interface-compatible local stubs for `pgx`, `pgxpool`, and `pgconn`.
- Existing Customer and Quote service tests passed in that static harness.
- Availability input-normalization tests passed.
- `go vet` passed in the same harness.
- Route registrations and handler signatures were checked.
- Migration files are embedded and ordered as:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
```

### Frontend

- All TypeScript and TSX files passed a strict local type-check harness with framework declarations for React and Next.js.
- New `/reservations` and `/reservations/[id]` route files were included.
- Quote detail contracts were updated for availability and reservation linkage.
- Internal imports and aliases were checked for resolvable project files.
- JSON configuration files were parsed.
- CSS block delimiters were balanced.
- Booking-specific CSS declarations, shared semantic hooks, and responsive rules were reviewed.

### Packaging

- `.env` is excluded from the upgrade archive.
- Generated build directories and dependency directories are excluded.
- No private key, token, password file, or copied local database is included.
- The full archive contains the entire project under one `rentstage-starter/` directory.
- The upgrade archive contains only files added or changed since v0.2.0.
- SHA-256 checksums are generated for both archives.

## Runtime validation still required locally

The packaging environment does not provide a Docker daemon, a PostgreSQL server, or online dependency resolution. Therefore, these checks must be completed on the user workstation:

1. Build the real Go image with Go 1.26.5 and `pgx v5.10.0`.
2. Build the real Next.js 16.3.0 production bundle.
3. Apply `003_booking_core.sql` to PostgreSQL 18.4.
4. Convert an accepted quote into a reservation.
5. Verify overlapping availability and cancellation release.
6. Verify every reservation transition.
7. Confirm PostgreSQL constraints and transaction behavior under concurrent conversion requests.

## Recommended local commands

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a

docker compose logs --tail=200 db api web

Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

## Critical concurrency scenario

Prepare two accepted quotes that each request all remaining units of the same resource during the same interval. Trigger both conversion requests as close together as possible.

Expected outcome:

```text
Request A → 201 reservation created
Request B → 409 availability_conflict
```

The final reserved quantity must never exceed eligible physical assets through the supported conversion endpoint.
