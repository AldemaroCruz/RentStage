# Upgrade RentStage v0.2.0 → v0.3.0

This update adds the Availability and Booking Cores without deleting the existing PostgreSQL volume.

## What is preserved

The upgrade keeps:

- `.env`, including a custom `POSTGRES_PORT=5433` value.
- Categories, resources, and physical assets.
- Customers.
- Existing quotes and their historical prices.
- Audit events.
- The Docker volume `rentstage_postgres_data`.

## What is added

- Migration `003_booking_core.sql`.
- Bulk availability checks.
- Transactional quote-to-reservation conversion.
- Reservation state machine and status history.
- Reservation list and detail screens.
- Quote links to generated reservations.
- Booking Core audit events.

## 1. Back up the project directory

From PowerShell:

```powershell
cd C:\Users\itres\Documents
Copy-Item rentstage-starter rentstage-starter-v0.2-backup -Recurse
```

For a database-level backup as well:

```powershell
cd C:\Users\itres\Documents\rentstage-starter
cmd /c "docker compose exec -T db pg_dump -U rentstage -d rentstage > rentstage-before-v0.3.sql"
```

When custom database credentials are configured in `.env`, substitute the appropriate user and database name.

## 2. Stop the current stack

```powershell
cd C:\Users\itres\Documents\rentstage-starter
docker compose down
```

Do **not** use:

```powershell
docker compose down -v
```

The `-v` option removes the PostgreSQL volume and is not part of this upgrade.

## 3. Overlay the upgrade package

Extract:

```text
rentstage-upgrade-v0.3.0.zip
```

directly into:

```text
C:\Users\itres\Documents\rentstage-starter
```

Allow Windows to replace existing files. The upgrade archive does not contain `.env`.

Confirm the version:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.3.0
```

## 4. Review the resolved Compose configuration

```powershell
docker compose config
```

When port 5432 is already used in Windows, confirm `.env` still contains:

```env
POSTGRES_PORT=5433
```

The internal API connection must continue using `db:5432`.

## 5. Rebuild API and web images

```powershell
docker compose build --no-cache api web
```

Then start the stack:

```powershell
docker compose up -d
docker compose ps -a
```

Expected services:

```text
rentstage-db-1    Up (healthy)
rentstage-api-1   Up (healthy)
rentstage-web-1   Up (healthy)
```

The API applies `003_booking_core.sql` automatically at startup.

## 6. Validate health

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

Expected statuses:

```text
ok
ready
```

Review startup logs:

```powershell
docker compose logs --tail=200 db api web
```

The API should start without a migration error.

## 7. Confirm migration state

```powershell
docker compose exec db psql -U rentstage -d rentstage -c "SELECT filename, applied_at FROM schema_migrations ORDER BY filename;"
```

Expected migration files include:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
```

Inspect the new reservation columns:

```powershell
docker compose exec db psql -U rentstage -d rentstage -c "\d reservations"
```

Important fields include:

```text
block_start_at
block_end_at
event_start_at
event_end_at
subtotal
discount_amount
extra_charges
total
```

## 8. End-to-end Booking Core test

Open:

```text
http://127.0.0.1:3000/quotes
```

With demo data enabled, use the sent quote `QT-000002`:

1. Open the quote.
2. Select **Aceptar**.
3. Select **Validar disponibilidad**.
4. Confirm every line shows sufficient stock.
5. Select **Crear reserva**.
6. Confirm RentStage redirects to an `RS-xxxxxx` detail page.
7. Open the reservations list:

```text
http://127.0.0.1:3000/reservations
```

The reservation should initially be `Pendiente` and block its quantities.

## 9. Validate inventory conflict behavior

Create or reuse another quote that requests more units than remain available during the same blocked interval.

After sending and accepting it, select **Crear reserva**. Expected behavior:

- API response: HTTP `409`.
- UI: availability conflict with the affected resource.
- No partial reservation is written.
- Existing reservations remain unchanged.

To inspect quantities directly:

```powershell
docker compose exec db psql -U rentstage -d rentstage -c "SELECT reservation_number, status, block_start_at, block_end_at FROM reservations ORDER BY reservation_number;"
```

## 10. Validate stock release

Open the first reservation and select **Cancelar reserva** while it is still in an allowed pre-checkout state.

Re-run availability for the same quote period. The cancelled reservation must no longer reduce available quantity.

The same release behavior applies after the reservation reaches `RETURNED` or `COMPLETED`.

## 11. Validate the state machine

For a non-cancelled reservation, use the visible actions in this order:

```text
PENDING
  → CONFIRMED
  → PREPARING
  → READY
  → CHECKED_OUT
  → RETURNED
  → COMPLETED
```

The detail page should append one timeline entry per transition. Invalid state jumps return HTTP `409`.

## API smoke checks

Set IDs from your local data before running these requests.

```powershell
$Tenant = "00000000-0000-0000-0000-000000000001"
$Headers = @{ "X-Tenant-ID" = $Tenant; "X-Actor-ID" = "local-admin" }

Invoke-RestMethod `
  -Method Get `
  -Uri "http://127.0.0.1:8080/api/v1/reservations" `
  -Headers $Headers
```

The full payload examples are in `docs/requests.http`.

## Troubleshooting

### API does not become healthy

```powershell
docker compose logs --tail=250 api
```

A migration error will appear before the API starts listening.

### Database container is not healthy

```powershell
docker compose logs --tail=250 db
```

Confirm the host port in `.env` does not conflict with another Windows service.

### UI still appears to be v0.2

Force a rebuild and recreate the web container:

```powershell
docker compose build --no-cache web
docker compose up -d --force-recreate web
```

Then hard-refresh the browser with `Ctrl+F5`.

### Need to roll back application files

Stop the stack and restore the copied v0.2 directory. The v0.3 database migration is forward-only in this starter. Restoring the database dump is the clean rollback when reverting schema as well.

```powershell
docker compose down
```

Do not delete the backup or SQL dump until the v0.3 validation is complete.
