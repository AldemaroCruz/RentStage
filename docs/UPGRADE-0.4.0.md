# Upgrade to RentStage v0.4.0

This package upgrades a working v0.3.0 installation to Warehouse Operations without deleting the existing PostgreSQL volume.

## What is preserved

- `.env`, including a custom `POSTGRES_PORT=5433`.
- PostgreSQL volume and all existing tenants, inventory, customers, quotes, and reservations.
- Existing migrations `001`, `002`, and `003`.

## Before upgrading

Confirm the current project:

```powershell
cd C:\Users\itres\Documents\rentstage-starter
Get-Content .\VERSION
docker compose ps -a
```

The expected starting version is:

```text
0.3.0
```

Creating a backup is recommended:

```powershell
docker compose exec -T db pg_dump `
  -U rentstage `
  -d rentstage `
  --format=custom `
  --file=/tmp/rentstage-before-v0.4.dump

docker compose cp db:/tmp/rentstage-before-v0.4.dump .\rentstage-before-v0.4.dump
```

## Apply the package

1. Stop the stack without deleting volumes:

```powershell
docker compose down
```

2. Extract `rentstage-upgrade-v0.4.0.zip` directly into:

```text
C:\Users\itres\Documents\rentstage-starter
```

Allow Windows to replace existing files.

3. Confirm the version:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.4.0
```

4. Rebuild and start:

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a
```

Do **not** use:

```powershell
docker compose down -v
```

The `-v` flag deletes the local PostgreSQL volume.

## Verify health

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

Expected statuses:

```text
ok
ready
```

Check logs:

```powershell
docker compose logs --tail=200 db api web
```

## Verify migration 004

```powershell
docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations ORDER BY version;"
```

Expected migration filenames:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
```

Optional schema checks:

```powershell
docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "\d reservation_assets"

docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "\d reservation_activity_events"
```

## Functional validation

The reservation completed during the v0.3 test cannot be moved back to preparation. Create a new reservation or convert a different accepted quote.

Recommended scenario:

1. Create or open a fresh quote containing:
   - `1 × Behringer X32 Compact`
   - `4 × Shure SM58`
2. Send, accept, validate, and convert it into a reservation.
3. Advance it to `PREPARING`.
4. Assign:
   - `MIX-X32-001`
   - `MIC-SM58-001`
   - `MIC-SM58-002`
   - `MIC-SM58-003`
   - `MIC-SM58-004`
5. Before assigning the fourth microphone, try **Marcar lista**. Expected: an inventory-incomplete conflict.
6. Finish assignment and mark the reservation `READY`.
7. Register check-out with delivery notes.
8. Return all five units. Set one microphone to `MAINTENANCE_REQUIRED` with a note.
9. Complete the reservation.
10. Open Inventory and confirm that the selected microphone is now `MAINTENANCE`; good units remain `AVAILABLE`.
11. Review both status history and warehouse activity history.

## Double-assignment validation

Prepare two overlapping reservations that both request the same model. Put both in `PREPARING` and attempt to assign the same exact asset.

Expected:

```text
First reservation  → assignment created
Second reservation → HTTP 409 asset_assignment_conflict
```

The response identifies the conflicting reservation.

## Common recovery commands

Rebuild only the API:

```powershell
docker compose build --no-cache api
docker compose up -d api
```

Rebuild only the frontend:

```powershell
docker compose build --no-cache web
docker compose up -d web
```

Inspect API logs:

```powershell
docker compose logs --tail=250 api
```

Inspect the running version:

```powershell
Get-Content .\VERSION
```
