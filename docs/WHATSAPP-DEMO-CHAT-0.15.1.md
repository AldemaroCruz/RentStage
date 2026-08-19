# WhatsApp demo conversation v0.15.1

RentStage v0.15.1 completes the commercial conversation simulator without pretending that Meta or a real phone number is connected. It is designed for a live proof of concept in which the presenter can show the same human-controlled workflow that a production provider will later drive.

## Demonstrable flow

1. Simulate a new customer inquiry from the assistant page.
2. Review the deterministic recommendation built from tenant packages, prices, and availability.
3. Select an existing customer or create one from the chat. New records use source `WHATSAPP`, the contact phone is prefilled, normal customer validation runs, and the conversation is linked within the current tenant.
4. Approve the commercial proposal and create a quote in `DRAFT`.
5. Review or edit the pending response and select **Enviar respuesta demo**.
6. Simulate another customer message, such as a price, availability, package, or payment question.
7. Review the new assistant draft and deliver another response in the simulator.

The cycle in steps 6–7 can be repeated. The transcript distinguishes inbound simulation, unsent drafts, approved pending messages, and simulated delivery.

## Safety boundary

- `send-demo` and `receive-demo` reject any conversation whose channel is not `DEMO`.
- “Delivered” means visible in the RentStage simulator only. No network request targets Meta and no telephone receives a message.
- Every assistant follow-up is created with status `DRAFT` and returns the conversation to `HUMAN_REVIEW`.
- Sending requires `assistant.manage`; reading remains available through `assistant.read`.
- Customer creation uses the existing `customer.manage` endpoint and audit event. Linking verifies that the customer belongs to the active tenant.
- Quote approval remains idempotent, creates only `DRAFT`, and never reserves inventory.

## Production channel — separate next increment

The production implementation should preserve these domain contracts and replace only the provider boundary. It must include:

- a verified Meta Business portfolio, WhatsApp Business Account, and dedicated sender number;
- Cloud API webhook verification, signature validation, idempotency for inbound message IDs, and delivery-status ingestion;
- access tokens stored in Secret Manager with rotation and least-privilege runtime access;
- opt-in evidence, opt-out enforcement, the 24-hour customer-service window, and approved template messages outside that window;
- an outbound queue with retries, rate limiting, dead-letter handling, and per-message cost/quality telemetry;
- a production adapter that maps Meta messages into the existing tenant-scoped conversation tables without bypassing RBAC, audit, quote, or inventory rules;
- staging test-number validation before enabling a real sender for customers.

No Meta resource or charge is created by v0.15.1.
