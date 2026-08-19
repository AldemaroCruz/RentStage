# WhatsApp demo → customer quote decision v0.15.3

## Presenter flow

1. Open **WhatsApp AI** and use **Reiniciar demo**.
2. Simulate a customer inquiry and review the deterministic package recommendation.
3. Link or create the customer, edit the response, and approve it. RentStage creates only a quote in `DRAFT`.
4. Deliver the reviewed chat response inside the simulator.
5. Select **Enviar cotización y generar portal** and open the secure customer link.
6. Review the quote as the customer and explicitly accept or reject it.
7. Return to the assistant. The portal view count and decision update automatically or through **Actualizar vistas y decisión**.

This sequence demonstrates the production business boundary without pretending that Meta is connected. No real phone receives a message.

## Credential handling

- The Quote Portal generates 32 random bytes and exposes them as a URL-safe bearer token in `/q#token`.
- The fragment is not sent in the browser's initial HTTP request. The Quote Portal moves it into `sessionStorage`, clears the visible fragment, and sends it only in `X-RentStage-Quote-Token`.
- The database stores only a SHA-256 digest. Issuing a replacement revokes the previous token.
- The assistant's issuance response uses `no-store` and includes the raw URL once. The admin browser retains it only in the current tab's `sessionStorage`.
- Chat messages and audit records store quote number, portal revision, simulated-delivery flags, and `raw_token_persisted=false`; they never store the URL or raw token.

## Human and inventory boundary

The rule-based demo prepares drafts. A signed-in operator must approve and deliver each response. The public customer must make the quote decision. Only the existing Quote Portal acceptance transaction can create a `PENDING` reservation after rechecking availability; rejection creates no reservation.

## Deferred production channel

A production WhatsApp increment still requires a verified Meta Business portfolio and sender, webhook signature verification, encrypted provider configuration, opt-in evidence, template lifecycle, the 24-hour customer-service window, delivery/read status ingestion, idempotent inbound event processing, rate limits, observability, and a human handoff/stop mechanism. Those provider concerns must call the existing assistant and quote application services rather than write business tables directly.
