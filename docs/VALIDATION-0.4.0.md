# Validation record — RentStage v0.4.0

## Completed before packaging

### Backend

- `gofmt` applied to all modified and new Go files.
- All RentStage Go packages compiled and tests ran against local interface-compatible declarations for `pgx`, `pgxpool`, and `pgconn`.
- Existing Customer, Quote, and Availability tests passed in the same harness.
- New Warehouse tests passed for return-input normalization, duplicate inspection detection, supported conditions, and return-to-physical-status mapping.
- `go vet` passed in the local harness.
- New route registrations and handler signatures were checked.
- Migration files are embedded and ordered as:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
```

### Frontend

- All TypeScript and TSX files passed syntax transpilation.
- The project passed a strict local type-check harness with declarations for React and Next.js.
- Reservation detail, warehouse panel, check-out modal, and return-inspection contracts were checked together.
- Internal aliased imports were checked for resolvable project files.
- JSON and Compose YAML were parsed.
- CSS block delimiters were balanced.

### Packaging

- `.env` is excluded from the upgrade archive.
- Generated `.next`, `node_modules`, temporary validation files, and local dependency stubs are excluded.
- No private key, token, password file, database dump, or copied local volume is included.
- The full archive contains the project under one `rentstage-starter/` directory.
- The upgrade archive contains only files added or changed since v0.3.0.
- Overlaying the upgrade archive onto v0.3.0 is compared byte-for-byte with the full v0.4.0 tree.
- SHA-256 checksums are generated and verified for both archives.

## Runtime validation still required locally

The packaging environment does not provide a Docker daemon, a PostgreSQL 18 server, or online dependency resolution. The workstation must perform the final integration checks:

1. Build the real Go 1.26.5 image with `pgx v5.10.0`.
2. Build the real Next.js 16.3.0 production bundle.
3. Apply `004_warehouse_operations.sql` to the existing PostgreSQL volume.
4. Assign and unassign exact assets during `PREPARING`.
5. Confirm that incomplete preparation cannot become `READY`.
6. Confirm overlapping reservations cannot use the same physical asset.
7. Check out all assignments.
8. Return every unit with inspection results.
9. Confirm physical-status changes in Inventory.
10. Confirm cancellation releases assignments.

## Recommended commands

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a

docker compose logs --tail=200 db api web

Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

## Critical assignment-concurrency scenario

Prepare two overlapping reservations and issue assignment requests for the same physical asset as close together as possible.

Expected result:

```text
Request A → 201 assignment created
Request B → 409 asset_assignment_conflict
```

The same unit must never be assigned to two overlapping blocking reservations through the supported warehouse endpoint.
