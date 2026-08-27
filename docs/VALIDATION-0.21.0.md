# RentStage v0.21.0 validation record

Validation date: 2026-08-27

## Completed locally

- [x] Go formatting is clean and all modules verify with Go 1.26.6.
- [x] Full API tests pass with the race detector, shuffled execution, and a fresh count.
- [x] `go vet ./...` passes.
- [x] Web type checking, unit tests, coverage thresholds, and production build pass with Node 24.
- [x] Docker Compose configuration resolves and the local stack starts successfully in credential-free `rules` mode.
- [x] The complete authenticated and public smoke suite passes, including packages, assistant, quote portal, billing, and MOCK DTE invalidation.
- [x] `git diff --check` passes and the feature checkpoints leave a clean worktree.

## AI and grounding checks

- [x] Initial and follow-up drafts use bounded visible conversation history.
- [x] Public packages, visible resources, allowed prices, and public settings reach Vertex as a bounded catalog snapshot.
- [x] Hidden resources and hidden prices are excluded.
- [x] Successful Vertex drafts persist canonical `PACKAGE` and `RESOURCE` evidence.
- [x] Reviewer evidence and the structured sales brief render only in the authenticated assistant inbox.
- [x] Sales signals are literal customer-message fragments; `TEAM` content and invented values are rejected.
- [x] Client-message retries remain idempotent with one inbound message and one assistant draft.
- [x] A one-second Vertex timeout produces deterministic fallback with `TIMEOUT`.
- [x] A 20-second retry succeeds through `gemini-2.5-flash` without fallback.
- [x] Prompt injection requesting a confirmed reservation, guaranteed availability, and an ungrounded `999 USD` price is rejected as `INVALID_RESPONSE`.
- [x] Invalid fallback output clears grounding references and sales-brief signals.

## Privacy and human control

- [x] Public session responses contain only published outbound messages and customer inbound messages.
- [x] Public responses expose no assistant draft, metadata, grounding evidence, sales brief, model, fallback reason, or raw provider diagnostic.
- [x] Every generated response remains `DRAFT` with `human_approval_required=true` until explicit authenticated publication.
- [x] No test creates an autonomous quote, reservation, inventory mutation, payment, DTE, Meta delivery, or other external action.
- [x] Temporary ADC mounts are removed after Vertex tests and the local runtime returns to `ASSISTANT_AI_MODE=rules`.

## Pull-request gates pending

- [ ] Repository validation.
- [ ] API unit, race, and vet.
- [ ] Web unit, coverage, typecheck, and build.
- [ ] Terraform staging and production validation.
- [ ] Gitleaks, govulncheck, gosec, npm audit, Trivy, and dependency review.
- [ ] CodeQL for Go and JavaScript/TypeScript.
- [ ] Docker integration and container-image scans.

No staging or production deployment is required to approve the pull request. Deployment remains governed by the existing repository and environment gates.
