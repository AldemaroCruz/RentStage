# Meta WhatsApp local development in RentStage v0.18.0

This increment implements the application contract needed for a future WhatsApp Cloud API connection without requiring a Meta account, a business number, a public callback URL, or paid message delivery. Nothing in this mode can contact a real telephone.

## Local boundary

Docker Compose enables `META_WHATSAPP_MODE=local_mock` only for the local API container. Configuration validation rejects this mode unless:

- `APP_ENV` is `local`;
- the Graph base URL uses plain HTTP on `127.0.0.1`, `localhost`, or `::1`;
- the URL ends in `/api/v1/integrations/meta/local-graph`;
- all identifiers and placeholder secrets are present.

Staging and production do not inject these variables, so their default remains `disabled`. The deterministic values in `.env.example` are test fixtures, not credentials.

## Implemented contract

| Path | Purpose | Protection |
| --- | --- | --- |
| `GET /api/v1/integrations/meta/webhook` | Subscription verification | constant-time verify-token comparison; raw challenge response |
| `POST /api/v1/integrations/meta/webhook` | Inbound messages and status callbacks | 1 MiB limit; exact-body `X-Hub-Signature-256` validation before JSON parsing |
| `POST /api/v1/integrations/meta/local-graph/{version}/{phoneNumberID}/messages` | Local text delivery substitute | loopback-only configuration, isolated bearer token, expected phone-number ID |

Inbound text messages are mapped to the existing `WHATSAPP` assistant channel. The provider message ID is unique per tenant and makes webhook retries idempotent. A valid customer message extends `service_window_expires_at` by 24 hours and creates a human-review draft. Sending is blocked after that window. Non-text messages are counted as ignored rather than silently interpreted.

The database stores the tenant association, mode, phone-number ID, WABA ID, and display number. It never stores the access token, application secret, or webhook verification token.

## Start and exercise the harness

```powershell
Set-Location C:\Users\itres\Documents\rentstage-starter

docker compose up --build --detach
bash scripts/ci/wait-local.sh

pwsh `
  -NoLogo `
  -NoProfile `
  -ExecutionPolicy Bypass `
  -File .\scripts\smoke-meta-local.ps1
```

Then open `http://127.0.0.1:3000/assistant`. The signed smoke event creates the **Cliente Meta Local** conversation. Review its draft and send a response; the UI records a `wamid.local.*` identifier but no external delivery occurs.

## Production parity and intentional gaps

The harness deliberately mirrors the stable boundaries needed later: webhook verification, signed callbacks, official message/status envelopes, Graph text-send shape, provider IDs, retry idempotency, human approval, and the 24-hour customer-service window.

It does not create a Meta Business Portfolio, app, WABA, system user, access token, sender, approved template, public HTTPS callback, webhook subscription, billing arrangement, or production credential version. Those steps belong to a separately reviewed production increment after infrastructure and cost approval.

Official references used for this contract:

- [Receiving messages (archived official WhatsApp SDK documentation)](https://whatsapp.github.io/WhatsApp-Nodejs-SDK/receivingMessages/)
- [Sending text messages (archived official WhatsApp SDK documentation)](https://whatsapp.github.io/WhatsApp-Nodejs-SDK/api-reference/messages/text/)
- [Official WhatsApp Node.js SDK repository and archive notice](https://github.com/WhatsApp/WhatsApp-Nodejs-SDK)
