# Upgrade seguro a RentStage v0.2.0

Esta actualización agrega Customer Core y Quote Core sin borrar el inventario existente.

## 1. Conserva tu configuración

No reemplaces tu archivo `.env`. En particular, si Windows ya utiliza PostgreSQL 5432, conserva:

```env
POSTGRES_PORT=5433
```

La API seguirá conectándose internamente a `db:5432`.

## 2. Detén los contenedores sin borrar el volumen

Desde PowerShell, dentro del proyecto:

```powershell
docker compose down
```

No uses `-v`.

## 3. Copia la actualización

Descomprime `rentstage-upgrade-v0.2.0.zip` sobre:

```text
C:\Users\itres\Documents\rentstage-starter
```

Permite que Windows reemplace los archivos existentes. El paquete no contiene `.env`.

## 4. Reconstruye API y frontend

```powershell
docker compose build --no-cache api web
docker compose up -d
docker compose ps -a
```

La API aplicará automáticamente `002_customer_quotes.sql` y conservará el volumen `rentstage_postgres_data`.

## 5. Verifica salud y migración

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz

docker compose logs --tail=150 db
docker compose logs --tail=150 api
docker compose logs --tail=150 web
```

Resultados esperados:

```text
/healthz → status: ok
/readyz  → status: ready
```

En los logs de API no debe aparecer `database migration failed`.

## 6. Abre los módulos nuevos

```text
http://127.0.0.1:3000/customers
http://127.0.0.1:3000/quotes
```

Con `SEED_DEMO_DATA=true` aparecerán dos clientes y dos cotizaciones de ejemplo. El seed es idempotente.

## Recuperación rápida

Si la compilación usa capas viejas:

```powershell
docker compose down --remove-orphans
docker compose build --no-cache api web
docker compose up -d
```

Si algo falla, recopila:

```powershell
docker compose ps -a
docker compose logs --tail=250 db api web
```

No ejecutes `docker compose down -v` salvo que quieras eliminar deliberadamente todos los datos locales.
