# Meta WhatsApp production roadmap

The current RentStage assistant is a safe `DEMO` provider. The production channel must call the same tenant-scoped assistant, customer, quote, portal, reservation, and audit services rather than write their tables directly.

The authoritative setup references are the Meta developer documentation for [WhatsApp Cloud API](https://developers.facebook.com/docs/whatsapp/cloud-api/), [webhooks](https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/), and [message templates](https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates/). Confirm the current requirements in those references during onboarding because Meta product names, review steps, API versions, limits, and pricing can change.

## Phase 1 — Production boundary and Meta account

1. Finish the separate production GCP project and reviewed infrastructure apply.
2. Create and verify the Meta business portfolio and app.
3. Create or attach the production WhatsApp Business Account and dedicated sender number.
4. Grant only the assets and permissions required by a dedicated system user.
5. Record the app ID, WABA ID, phone-number ID, app secret, access token, and random webhook verification token in the existing production Secret Manager containers.
6. Define token ownership, rotation, revocation, emergency disablement, and audit evidence before sending a customer message.

No credential belongs in Terraform variables, Terraform state, GitHub logs, build arguments, browser code, screenshots, or the RentStage database.

## Phase 2 — Inbound webhook adapter

- Expose a narrow public verification endpoint and a POST event endpoint.
- Validate the verification challenge without exposing the stored verification token.
- Verify every POST signature with the Meta app secret before parsing business data.
- Enforce payload size, content type, request timeout, rate limits, and structured redacted logging.
- Persist the provider event/message ID under a uniqueness constraint before acknowledging it, so retries are idempotent.
- Resolve tenant and sender from server-owned provider configuration, never from a browser-supplied tenant ID.
- Queue accepted events and return quickly; business logic runs asynchronously with retry and dead-letter evidence.

## Phase 3 — Human-controlled outbound adapter

- Keep the current editable draft and explicit human approval.
- Validate E.164 destination, tenant ownership, opt-in evidence, conversation status, and the applicable service-window/template policy immediately before sending.
- Use free-form replies only where the current WhatsApp customer-service rules allow them; otherwise require a currently approved template.
- Store the sanitized request, Meta message ID, Graph API version, template identity/version, approving actor, and result.
- Never treat an HTTP success as delivery. Update sent, delivered, read, and failed states from signed webhook status events.
- Provide visible retry, cancel, handoff, stop-contact, and provider-disable controls.

## Phase 4 — Pilot and operations

- Start with the business owner's number and a small allowlist of consented testers.
- Separate test and production WABA/sender assets when Meta permits the target setup.
- Alert on webhook signature failures, duplicate storms, send failures, token errors, queue age, and delivery latency.
- Build dashboards for inbound/outbound/status counts without logging message bodies or credentials.
- Document privacy retention, deletion requests, opt-out handling, incident response, and cost ownership.
- Expand beyond the allowlist only after message templates, business display, consent language, support ownership, and failure recovery are accepted.

## Release sequence

1. `v0.17.x`: production project, plan, approval, and recovery foundation.
2. Provider configuration migration plus Secret Manager loader with no send path.
3. Signature-verified/idempotent inbound webhook and status ingestion.
4. Allowlisted human-approved outbound text/template pilot.
5. Production Cloud Run deployment, observability, runbook, and controlled customer onboarding.

The production switch should be a tenant/provider configuration decision with a global emergency off switch. `DEMO` must remain available for presentations and must never share credentials or message IDs with the Meta provider.
