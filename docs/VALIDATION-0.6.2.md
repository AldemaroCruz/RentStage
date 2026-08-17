# RentStage v0.6.2 validation record

## Scope

Patch-only validation for the Next.js production prerender failure introduced by the v0.6 root route guard.

## Validated

- The reported stack points to `components/RootFrame.tsx` while prerendering `/_not-found`.
- `Frame` is the component that calls `useSearchParams()`.
- `Frame` and its `children` now have an ancestor React `Suspense` boundary.
- `/login` and `/signup`, whose page components also call `useSearchParams()`, render below the same boundary.
- The fallback uses existing RentStage CSS classes and introduces no new styling dependency.
- TypeScript/TSX syntax transpilation passes for the complete `apps/web/app`, `apps/web/components`, and `apps/web/lib` trees.
- No Go source, SQL migration, Compose service, authentication-emulator image, API route, environment contract, or persisted-volume definition changed.
- No `.env`, database dump, `node_modules`, `.next`, or local secret was packaged.
- Applying the hotfix to the v0.6.1 tree produces the same changed files as the full v0.6.2 tree.

## Environment limitation

The final Docker build was not executed in the artifact environment because Docker Engine is unavailable and the npm registry cannot be reached from that environment. Complete the integration check on the Windows host with:

```powershell
docker compose build --no-cache web
docker compose build api
docker compose up -d
docker compose ps -a
```

The expected web-build result is successful generation of all static pages, including `/_not-found`.
