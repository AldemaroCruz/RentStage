# RentStage v0.6.2 Next.js prerender hotfix

## Symptom

The v0.6.1 web image compiles and passes TypeScript, but the production build stops while generating static pages:

```text
useSearchParams() should be wrapped in a suspense boundary at page "/404"
Error occurred prerendering page "/_not-found"
target web: failed to solve: process "/bin/sh -c npm run build" did not complete successfully
```

The Go build is shown as `CANCELED` only because Docker Compose cancels sibling build work after the web target fails. It is not the root cause.

## Root cause

`apps/web/components/RootFrame.tsx` is mounted from the root layout and therefore participates in every route, including Next.js' internal not-found route. Its `Frame` component calls `useSearchParams()` without an ancestor `Suspense` boundary. Next.js 16 requires that boundary during production prerendering.

The `/login` and `/signup` pages also consume `useSearchParams()`. Wrapping `Frame` and its `children` fixes all three call sites with one boundary.

## Fix

`RootFrame` now renders:

```tsx
<AuthProvider>
  <Suspense fallback={<FrameFallback />}>
    <Frame>{children}</Frame>
  </Suspense>
</AuthProvider>
```

## Apply over v0.6.1

1. Stop the stack without deleting volumes:

```powershell
docker compose down
```

2. Extract `rentstage-hotfix-v0.6.2.zip` directly into the project root and allow replacement of existing files.

3. Confirm the version:

```powershell
Get-Content .\VERSION
```

Expected:

```text
0.6.2
```

4. Rebuild the web image first:

```powershell
docker compose build --no-cache web
```

5. Build the API and start the complete stack:

```powershell
docker compose build api
docker compose up -d
docker compose ps -a
```

6. Verify the services:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
Invoke-WebRequest http://127.0.0.1:3000/login -UseBasicParsing
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

Do not use `docker compose down -v`. This hotfix has no migration and does not require resetting PostgreSQL or the Firebase emulator volume.
