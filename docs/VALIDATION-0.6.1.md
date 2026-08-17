# RentStage v0.6.1 validation record

## Scope

Patch-only validation for the Firebase Authentication emulator Docker image.

## Validated

- `apps/auth-emulator/Dockerfile` now uses `node:24-trixie-slim`.
- The Dockerfile installs `openjdk-21-jre-headless` from Debian Trixie repositories.
- Firebase CLI remains pinned to `15.26.0`.
- Build-time checks invoke `java -version` and `firebase --version`.
- The entrypoint, emulator ports, persistent volume, Compose healthcheck, API dependency, and environment contract are unchanged.
- No SQL migration was added.
- No `.env`, database dump, `node_modules`, `.next`, or local secret was packaged.
- Applying the hotfix to the original v0.6.0 tree produces the same changed files as the full v0.6.1 tree.

## Environment limitation

The final Docker build was not executed in the artifact environment because it has no Docker Engine. The integration check must be completed with:

```powershell
docker compose build --no-cache auth
docker compose up -d
docker compose ps -a
```
