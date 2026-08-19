# RentStage v0.15.3 validation record

## Repository and automated contracts

- [ ] `VERSION`, `apps/web/package.json`, README, dashboard, assistant hero, and changelog report `0.15.3`.
- [ ] Repository version, workflow YAML, migration ordering, sensitive-file, shell, and PowerShell syntax checks pass.
- [ ] Go formatting, module verification, unit/race tests, and vet pass.
- [ ] Frontend typecheck, unit coverage thresholds, and optimized production build pass.
- [ ] Docker integration, complete local smoke suite, source/dependency security, container scans, and CodeQL pass.
- [ ] Existing Terraform formatting and validation remain green.

## Assistant → Quote Portal behavior

- [ ] A human-approved assistant proposal creates only a `DRAFT` quote.
- [ ] Sharing changes the quote to `SENT` and returns `/q#token` exactly once with no-store headers.
- [ ] The token is present in neither serialized messages nor audit metadata; transcript evidence says `raw_token_persisted=false`.
- [ ] The current browser tab can open and copy the customer link; a page reload can restore it from `sessionStorage`, not persistent local storage.
- [ ] Rotating the link invalidates the previous token.
- [ ] Portal views and the customer decision appear after refresh/polling.
- [ ] Explicit rejection creates no reservation. Explicit acceptance uses the existing transactional Quote Portal acceptance flow.
- [ ] A user missing either `assistant.manage` or `quote.manage` cannot issue a link.

## Appearance behavior

- [ ] The top bar exposes an accessible light/dark mode button.
- [ ] Public/login/Quote Portal screens expose the same control without requiring authentication.
- [ ] The choice persists in `localStorage`, is applied before hydration, and follows the operating-system preference until a choice is stored.
- [ ] Light, dark, responsive, and print views retain readable contrast and usable controls.

## Evidence

```text
CI/CD run: ______________________________
Assistant smoke: ________________________
Frontend coverage: ______________________
Staging URL: ____________________________
Commit SHA: _____________________________
Validated by: ___________________________
Date: __________________________________
```
