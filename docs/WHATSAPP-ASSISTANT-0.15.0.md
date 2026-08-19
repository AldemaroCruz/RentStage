# WhatsApp Sales Assistant v0.15.0

RentStage v0.15.0 provides a commercial conversation flow before a Meta Business account or sender number exists. It is intentionally a product simulator, not a fake live integration.

## What is real

- Authentication, tenant isolation, roles, customers, packages, prices, inventory availability, quotes, and audit records use the normal RentStage backend.
- The recommendation engine evaluates active, ready packages and checks availability for the requested period.
- A manager, administrator, or owner chooses the customer, edits the response, and approves the proposal.
- Approval creates a normal quote with status `DRAFT` and records the approving actor.

## What is simulated

- `channel=DEMO` represents a WhatsApp-style incoming message; no phone receives or sends data.
- `provider=DEMO_RULES` is deterministic so the CONAMYPE presentation works without credentials, network access, model quota, or nondeterministic text.
- Consent is labeled `DEMO`; it must not be interpreted as customer opt-in for a future production channel.

## Non-negotiable control boundary

The assistant may recommend and draft. It does not send a message, accept a quote, create or confirm a reservation, assign an asset, issue an invoice, or mutate availability. Quote creation requires `assistant.manage`, and the resulting quote remains `DRAFT`.

## Future Meta adapter

A production adapter should preserve the current domain contract and add:

1. Meta webhook signature verification and idempotent external message IDs.
2. Explicit opt-in evidence, privacy/retention policy, and opt-out handling.
3. The 24-hour customer-service window plus approved templates outside that window.
4. Sent/delivered/read/failed status reconciliation.
5. Human escalation, rate limiting, retry policy, and operational monitoring.
6. Secret Manager references and keyless deployment; no credentials in GitHub or source control.

The database already distinguishes `DEMO` and `WHATSAPP`, stores external IDs, consent, the service-window expiry, provider evidence, and approval metadata.
