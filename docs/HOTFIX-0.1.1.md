# RentStage API dependency hotfix — v0.1.1

This revision fixes the Docker build error:

```text
missing go.sum entry for module providing package github.com/jackc/pgx/v5/...
```

## Changes

- Adds `apps/api/go.sum`.
- Records the indirect modules required by `pgx/v5` in `apps/api/go.mod`.
- Copies both module files before dependency download so Docker can reuse the layer cache.
- Verifies downloaded modules with `go mod verify`.
- Compiles with `-mod=readonly`, preventing silent module-file changes during the final build.

## Rebuild

From the repository root:

```powershell
docker compose down
docker compose build --no-cache api
docker compose up --build -d
docker compose ps
```

To rebuild everything from scratch, including the database volume:

```powershell
docker compose down -v --remove-orphans
docker compose up --build -d
```
