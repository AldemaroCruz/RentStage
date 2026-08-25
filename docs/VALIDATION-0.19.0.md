# RentStage v0.19.0 validation record

## Release consistency

- [ ] `VERSION`, web package, README title, dashboard label, assistant hero, and changelog report `0.19.0`.
- [ ] Repository contracts, workflow YAML, migration ordering, sensitive-file policy, and `git diff --check` pass.
- [ ] Migration ordering ends at `016_omnichannel_web_chat.sql` with no duplicate or missing number.
- [ ] Go formatting and all API tests pass.
- [ ] Frontend TypeScript, 41 focused tests, coverage thresholds, and the Next.js production build pass.
- [ ] API and web containers become healthy after a clean rebuild.

## Public-session contract

- [ ] Chat is absent when `web_chat_enabled` is false and present on every public-catalog route when true.
- [ ] Session creation requires a valid name, message, UUID, accepted consent, and empty honeypot.
- [ ] Optional email is normalized and invalid email is rejected.
- [ ] The raw token is returned only on creation, is 43 base64url characters, and PostgreSQL stores only its SHA-256 hash.
- [ ] Session retrieval requires tenant slug, session ID, and token; an invalid combination reveals no session details.
- [ ] Session responses are non-cacheable and contain no draft messages.
- [ ] Sessions expire after seven days and reject closed or expired access.
- [ ] Messages over 2,000 characters and more than 60 inbound messages per rolling hour are rejected.
- [ ] Repeating one client message UUID produces no duplicate message or draft.

## Human-controlled delivery

- [ ] Each inbound visitor message appears once in the authenticated inbox as `WEB_CHAT`.
- [ ] The operator sees contact name and optional email without requiring a telephone number.
- [ ] The assistant response begins as `DRAFT` and is invisible to the visitor.
- [ ] Only an authenticated authorized user can publish the draft.
- [ ] Publication records sender type `USER`, provider `WEB_CHAT`, and status `SENT`.
- [ ] Superseded unsent drafts are marked failed and never become public.
- [ ] The approved response appears only in the matching browser session within the polling interval.
- [ ] No assistant or chat action reserves inventory, approves a quote, or creates a production delivery side effect.

## Isolation and regression

- [ ] Demo conversations and the Meta local harness continue to send through their original paths.
- [ ] Meta cloud outbound remains blocked and no real phone is contacted.
- [ ] Public quote requests, Quote Portal, customer linking, and catalog publication still work.
- [ ] No token, secret, credential, or sensitive request body appears in logs or source control.
- [ ] No new GCP resource, production deployment, secret version, IAM binding, or Terraform apply is introduced.

## Accepted local evidence

- [ ] Visitor creates a web-chat session from `/p/audiopro-demo`.
- [ ] Inbox reports the new conversation as **Web chat · Human review**.
- [ ] An operator presses **Publish to web chat**.
- [ ] The approved response appears in the existing visitor widget without page reload.
