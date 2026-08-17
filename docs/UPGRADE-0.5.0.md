# Upgrade RentStage v0.4.0 → v0.5.0

This package upgrades an existing v0.4 installation without replacing its PostgreSQL volume.

## What is preserved

- `.env`, including `POSTGRES_PORT=5433` on Windows.
- Tenant and users.
- Categories, resources, and physical assets.
- Customers and quotes.
- Reservations and status history.
- Warehouse assignments, check-out records, return inspections, and activity history.
- Existing audit events.

## New migration

```text
005_calendar_operations.sql
```

It adds:

- `reservations.source`;
- reservation-source consistency constraints;
- `reservation_schedule_history`;
- calendar and source indexes.

Existing reservations are backfilled as:

```text
quote_id IS NOT NULL → QUOTE
quote_id IS NULL     → MANUAL
```

## 1. Optional but recommended backup

From PowerShell:

```powershell
cd C:\Users\itres\Documents\rentstage-starter
New-Item -ItemType Directory -Force .\backups | Out-Null

docker compose exec db `
  pg_dump `
  -U rentstage `
  -d rentstage `
  -Fc `
  -f /tmp/rentstage-v0.4.0-before-v0.5.dump

docker compose cp `
  db:/tmp/rentstage-v0.4.0-before-v0.5.dump `
  .\backups\rentstage-v0.4.0-before-v0.5.dump
```

## 2. Stop containers without deleting volumes

```powershell
docker compose down
```

Do not run:

```powershell
docker compose down -v
```

## 3. Extract the upgrade

Extract `rentstage-upgrade-v0.5.0.zip` directly into:

```text
C:\Users\itres\Documents\rentstage-starter
```

Allow replacement of existing files. The upgrade ZIP does not contain `.env`.

## 4. Confirm version

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.5.0
```

## 5. Rebuild and start

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a
```

Expected services:

```text
rentstage-db-1    healthy
rentstage-api-1   healthy
rentstage-web-1   healthy
```

## 6. Verify health

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

Expected:

```text
ok
ready
```

## 7. Confirm migrations

```powershell
docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT version, applied_at FROM schema_migrations ORDER BY version;"
```

Expected sequence:

```text
001_init.sql
002_customer_quotes.sql
003_booking_core.sql
004_warehouse_operations.sql
005_calendar_operations.sql
```

## 8. Verify source backfill

```powershell
docker compose exec db psql `
  -U rentstage `
  -d rentstage `
  -c "SELECT source, COUNT(*) FROM reservations GROUP BY source ORDER BY source;"
```

Existing quote-derived reservations should appear as `QUOTE`.

## 9. Functional test

1. Open http://127.0.0.1:3000/calendar.
2. Switch between Month, Week, and Agenda.
3. Open **Reservas → Nueva reserva**.
4. Select a customer and create a manual reservation after verifying availability.
5. Confirm the reservation appears in the calendar.
6. Open it and use **Reprogramar**.
7. Test one available and one conflicting period.
8. Confirm the **Reprogramaciones** history appears.
9. Review the alert bell and the Operations Center alert panel.

## Troubleshooting

### API unhealthy

```powershell
docker compose logs --tail=250 api
```

### Migration failure

```powershell
docker compose logs --tail=250 api db
```

The API runs embedded migrations before accepting traffic. A failed `005` migration leaves the previous migrations and user data intact because each migration is transactional.

### Frontend still shows v0.4 pages

```powershell
docker compose build --no-cache web
docker compose up -d --force-recreate web
```

### Restore the backup

Only use this after diagnosing the failed upgrade and intentionally replacing the database:

```powershell
docker compose down -v
docker compose up -d db
docker compose cp .\backups\rentstage-v0.4.0-before-v0.5.dump db:/tmp/restore.dump
docker compose exec db pg_restore -U rentstage -d rentstage --clean --if-exists /tmp/restore.dump
```
