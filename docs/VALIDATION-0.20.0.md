# RentStage v0.20.0 validation record

## Release consistency

- [ ] `VERSION`, web package, web lockfile, README title, dashboard label, assistant label, configuration guard, and changelog report `0.20.0`.
- [ ] Repository contracts, workflow YAML, migration ordering, sensitive-file policy, Compose configuration, and `git diff --check` pass.
- [ ] Migration ordering still ends at `016_omnichannel_web_chat.sql`.
- [ ] Go formatting, module verification, unit tests, and vet pass.
- [ ] Frontend TypeScript, focused tests, coverage thresholds, and the Next.js production build pass using `npm ci`.
- [ ] API and web production images build successfully from the committed lockfile.

## Provider selection and configuration

- [ ] The default configuration selects `rules` without requiring Google credentials.
- [ ] `vertex` requires a project, location, and model.
- [ ] Unsupported modes fail closed during startup.
- [ ] Timeout values outside 1–20 seconds are rejected.
- [ ] output-token values outside 64–2048 are rejected.
- [ ] Staging defaults to `rules` unless protected variables explicitly enable Vertex.
- [ ] The Cloud Run API runtime service account has `roles/aiplatform.user`; no JSON key is deployed.

## Draft generation

- [ ] An initial inbound web-chat message creates one private outbound assistant `DRAFT`.
- [ ] A follow-up inbound message creates one private outbound assistant `DRAFT`.
- [ ] A successful Vertex draft records `engine=VERTEX_AI`, the configured model, `used_fallback=false`, and `human_approval_required=true`.
- [ ] A deterministic draft records `engine=WEB_CHAT_RULES`, `model=DETERMINISTIC_V1`, and `used_fallback=false`.
- [ ] Provider errors and invalid or oversized output produce a valid deterministic draft with `used_fallback=true`.
- [ ] Draft bodies remain inside the public web-chat message-length boundary.

## Idempotency and isolation

- [ ] Retrying the same `client_message_id` creates one inbound message and one assistant draft.
- [ ] The draft metadata retains the matching `source_message_id`.
- [ ] No draft is returned by the public-session API.
- [ ] Tenant and session access checks remain unchanged.
- [ ] Raw session tokens remain tab-scoped and PostgreSQL stores only their SHA-256 hashes.

## Human review and UI provenance

- [ ] A pending Vertex draft displays `Vertex AI · <model>`.
- [ ] A pending rules draft displays `Deterministic rules`.
- [ ] A fallback draft displays `Safe fallback`.
- [ ] Unknown or missing provenance metadata does not create a misleading badge.
- [ ] The provenance badge appears only while the assistant message is a pending draft.
- [ ] Only an authenticated authorized user can edit and publish the response.
- [ ] Published responses appear in the matching visitor session.

## Regression and safety boundary

- [ ] No AI operation automatically publishes a response.
- [ ] No AI operation approves a quote, creates a reservation, or mutates inventory.
- [ ] Meta cloud delivery remains blocked in v0.20.0.
- [ ] The complete local integration smoke suite passes.
- [ ] Repository validation, API race/vet, web build, Terraform validation, CodeQL, dependency security, Docker integration, and container scanning pass in CI.

## Accepted implementation evidence — 2026-08-26

- [x] Real Vertex AI initial draft returned `VERTEX_AI / gemini-2.5-flash`, without fallback, and required human review.
- [x] Real Vertex AI follow-up draft preserved one inbound and one outbound record across a repeated client-message identifier.
- [x] The deterministic provider produced a pending draft with the correct reviewer-facing provenance badge.
- [x] The Vertex provider produced a pending draft with the correct reviewer-facing provenance badge.
- [x] Publishing remained an explicit human action in the assistant inbox.
- [x] The local API was returned to `rules` mode and the temporary ADC mount was removed after testing.
- [x] The web lockfile was generated, validated with `npm ci`, and wired into Docker, CI, security audit, Make, and version consistency checks.

## Final evidence

```text
Version consistency: ____________________
Migration ordering: _____________________
Go test/vet: ____________________________
Web test/build: _________________________
Docker build: ___________________________
Complete smoke suite: ___________________
Security and CodeQL: ____________________
Pull request checks: ____________________
```
