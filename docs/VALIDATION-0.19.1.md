
### `docs/VALIDATION-0.19.1.md`

```markdown
# RentStage v0.19.1 validation record

## Release consistency

- [ ] `VERSION`, web package, README title, dashboard label, assistant label, configuration guard, and changelog report `0.19.1`.
- [ ] Repository contracts, workflow YAML, migration ordering, sensitive-file policy, and `git diff --check` pass.
- [ ] Migration ordering still ends at `016_omnichannel_web_chat.sql`.
- [ ] Go formatting and all API tests pass.
- [ ] Frontend TypeScript, 49 focused tests, coverage thresholds, and the Next.js production build pass.
- [ ] API and web containers become healthy after rebuilding.

## Idempotent visitor retries

- [ ] The widget retains one `client_message_id` while retrying unchanged message content.
- [ ] Editing the message creates a new client message identifier.
- [ ] Parallel form submissions are blocked while an operation is active.
- [ ] A failed transport preserves the visitor's message in the composer.
- [ ] Retrying after connectivity returns creates one inbound message and one assistant draft.
- [ ] PostgreSQL reports `COUNT(*) = 1` for the accepted retry test message.

## Session recovery

- [ ] A temporary restoration failure does not clear a valid tab-scoped session token.
- [ ] The visitor can explicitly retry restoration without creating another conversation.
- [ ] Missing or expired sessions return the visitor safely to the initial form.
- [ ] Raw tokens remain exclusively in browser-tab `sessionStorage`.
- [ ] PostgreSQL stores only the session-token SHA-256 hash.
- [ ] The widget copy accurately describes the seven-day server limit and tab-scoped browser lifetime.

## Polling lifecycle

- [ ] Only one polling request can be in flight at a time.
- [ ] Polling pauses while the page is hidden.
- [ ] Returning to the visible tab triggers immediate synchronization.
- [ ] Opening an active widget triggers immediate synchronization.
- [ ] Temporary failures use bounded delays of 4, 8, 16, and 30 seconds.
- [ ] A successful request resets the failure counter and clears the connection notice.
- [ ] Closing the widget cancels the timer and any active polling request.
- [ ] A terminal `404` or `410` clears the unusable session and stops polling.

## Human-control and regression boundary

- [ ] Assistant drafts remain invisible to the visitor.
- [ ] Only an authenticated authorized user can publish a response.
- [ ] Approved replies still appear in the matching visitor session.
- [ ] No retry or polling operation approves a quote or reserves inventory.
- [ ] Meta cloud delivery remains blocked and no real telephone is contacted.
- [ ] No migration, external provider, WebSocket, service worker, secret, IAM binding, Terraform resource, or production deployment is introduced.

## Accepted local evidence

- [ ] Network interruption displays the localized reconnection notice without removing the loaded widget.
- [ ] Returning online restores polling and preserves the existing conversation.
- [ ] The message `Prueba de reintento v0.19.1` appears once in the visitor widget.
- [ ] PostgreSQL reports exactly one matching inbound `WEB_CHAT` message.