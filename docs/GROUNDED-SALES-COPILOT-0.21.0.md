# Grounded sales copilot — RentStage v0.21.0

RentStage v0.21.0 turns the private AI reply-draft boundary introduced in v0.20.0 into a grounded sales copilot. The model receives bounded conversation context and a validated snapshot of the tenant's public catalog, while the application remains the authority for persistence, evidence, and publication.

## Request boundary

Every initial or follow-up draft request can contain:

- The current customer message.
- Up to 12 recent `CUSTOMER` or `TEAM` messages, bounded to 8,000 Unicode code points in total.
- Up to eight published packages and twelve visible resources, bounded to 16,000 Unicode code points.
- Public currency, price-visibility, resource-visibility, and quote-request settings.

The catalog snapshot never contains inventory availability, private resources, hidden prices, session tokens, credentials, internal identifiers, or long-term customer memory. Conversation and catalog values are JSON encoded and explicitly treated as untrusted data.

## Grounding and evidence

Vertex returns structured references alongside the proposed reply. RentStage accepts at most five references and validates each one against the exact catalog snapshot sent for that draft.

Accepted reference kinds are:

- `PACKAGE`
- `RESOURCE`

Names are canonicalized to their published spelling. An invented, hidden, duplicate, unsupported, oversized, or ambiguous reference invalidates the complete provider result. The application then persists only the deterministic fallback with `fallback_reason=INVALID_RESPONSE`.

Validated evidence is stored in existing assistant-message metadata and rendered only in the authenticated assistant inbox. Public chat responses never contain metadata or private drafts.

## Structured sales brief

The provider can extract one signal for each of these fields:

- `EVENT_TYPE`
- `EVENT_DATE`
- `LOCATION`
- `GUEST_COUNT`
- `BUDGET`

Every signal value must be a literal substring of a `CUSTOMER` message. `TEAM` messages cannot serve as evidence. The provider may also identify missing fields and suggest one short follow-up question. Invalid, duplicate, unsupported, oversized, or ungrounded values invalidate the provider response.

The brief is reviewer-only context. It does not create or modify a customer, quote, reservation, inventory item, invoice, payment, or DTE.

## Commercial claim guardrails

After structured validation, RentStage applies deterministic checks to the free-text reply. A draft is rejected if it claims that availability, a reservation, a discount, a payment, a quote, or an order is confirmed or completed.

Monetary amounts must match the currency and visible price of a validated catalog reference. Guest counts must come from a customer message or the published name, description, or capacity of a referenced package. A failed check produces the safe deterministic fallback and clears references and sales-brief metadata.

These checks are defense in depth. Explicit human review and publication remain mandatory for every response.

## Provider outcomes

Private metadata distinguishes three allowlisted fallback reasons:

- `TIMEOUT`
- `PROVIDER_ERROR`
- `INVALID_RESPONSE`

Fallback details never include raw provider errors, prompts, tokens, credentials, or rejected provider output. Local development remains in `rules` mode by default and requires no ADC mount.

## Compatibility

v0.21.0 adds no migration, dependency, public endpoint, secret, IAM role, autonomous action, or external delivery channel. It reuses `assistant_messages.metadata` and the existing authenticated review workflow.
