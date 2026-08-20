# Meta application readiness in RentStage v0.18.1

This increment prepares the application contract while the Meta business account is under review. It does **not** send a real WhatsApp message.

## What is ready

- Signed webhook verification and exactly-once inbound processing from v0.18.0.
- Explicit `STOP`, `SALIR`, `CANCELAR`, `BAJA`, `NO MÁS MENSAJES`, and `DEJAR DE RECIBIR` handling.
- Explicit re-entry through `START`, `INICIAR`, `CONTINUAR`, or `ALTA`.
- Monotonic sent, delivered, read, and failed message states.
- Public `/privacy`, `/terms`, `/data-deletion`, and `/support` routes.
- A secret-free authenticated readiness endpoint at `/api/v1/integrations/meta/readiness`.
- Schema for a future reviewed Meta template catalog.

## Before submitting to Meta

1. Configure the GitHub Environment variable `RENTSTAGE_SUPPORT_EMAIL` with a real, monitored address. The web image receives it as `NEXT_PUBLIC_SUPPORT_EMAIL` at build time.
2. Publish the four legal/support URLs on a stable HTTPS domain.
3. Replace preliminary legal copy after review by the operating entity and counsel.
4. Prepare a screen recording showing human approval, opt-out, data deletion, and the stated use case.
5. Keep outbound cloud delivery disabled until the account, app, WABA, sender, templates, webhook, and policies are approved.

## Enforced gate

`META_OUTBOUND_ENABLED=true` works only in `local_mock`. The API refuses to start with cloud mode plus outbound enabled. This makes an accidental production send impossible in v0.18.1.
