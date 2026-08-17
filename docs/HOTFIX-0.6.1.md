# RentStage v0.6.1 auth-emulator build hotfix

## Symptom

The v0.6.0 Docker build stops while building the `auth` image:

```text
E: Package 'openjdk-21-jre-headless' has no installation candidate
target auth: failed to solve
```

The image used `node:24-bookworm-slim` (Debian 12), while OpenJDK 21 is not available from that release's standard package repositories.

## Fix

The authentication emulator now uses:

```dockerfile
FROM node:24-trixie-slim
```

Debian 13 (Trixie) provides `openjdk-21-jre-headless`, while retaining Node.js 24 for Firebase CLI 15.26.0.

The build also validates both runtimes:

```dockerfile
RUN ... \
    && java -version \
    && npm install --global "firebase-tools@${FIREBASE_TOOLS_VERSION}" ... \
    && firebase --version
```

## Apply over v0.6.0

1. Stop the stack without deleting volumes:

```powershell
docker compose down
```

2. Extract `rentstage-hotfix-v0.6.1.zip` over the project root and allow replacement.

3. Confirm the version:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.6.1
```

4. Rebuild the failed authentication image first:

```powershell
docker compose build --no-cache auth
```

5. Build the remaining services and start the stack:

```powershell
docker compose build api web
docker compose up -d
docker compose ps -a
```

6. Verify:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

Open:

```text
http://127.0.0.1:3000/login
http://127.0.0.1:4000
```

Local owner:

```text
owner@rentstage.local
RentStage123!
```

## Data safety

Do not use `docker compose down -v`. This hotfix does not require a database reset or a new migration.
