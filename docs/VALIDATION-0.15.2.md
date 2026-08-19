# RentStage v0.15.2 validation record

## Automated contracts

- [ ] `VERSION`, `apps/web/package.json`, README, dashboard release link, and changelog report `0.15.2`.
- [ ] `npm run typecheck` passes.
- [ ] All frontend unit tests pass.
- [ ] Coverage is at least 95% lines, 90% branches, and 95% functions.
- [ ] `format.ts` reports 100% lines, branches, and functions.
- [ ] The optimized Next.js production build passes.
- [ ] Repository contracts, API race tests, Docker smoke, security scans, and CodeQL remain green in the existing parallel workflow.

## Expected coverage

The current focused suite should report approximately:

```text
cloud-run-auth.ts  100% lines   91%+ branches  100% functions
format.ts          100% lines  100% branches   100% functions
all files           99%+ lines  96%+ branches  100% functions
```

Small runtime or Node/ICU differences may alter decimals, but the enforced thresholds must remain satisfied.

## Evidence

```text
CI/CD run: ______________________________
Coverage lines: _________________________
Coverage branches: ______________________
Coverage functions: _____________________
Commit SHA: _____________________________
Validated by: ___________________________
Date: __________________________________
```
