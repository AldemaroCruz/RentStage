# First-party web chat in RentStage v0.19.0

## Purpose

v0.19.0 introduces the first customer-facing channel on RentStage's omnichannel conversation core. A visitor can contact a tenant from the public catalog, while the tenant continues working from the existing assistant inbox and retains human control over every outbound response.

The feature works locally without Meta, a telephone number, a public webhook, a tunnel, a third-party chat provider, or a new cloud resource.

## Visitor flow

1. The public catalog exposes the widget only when both the catalog and `web_chat_enabled` are active.
2. The visitor provides a name, an optional email, an initial message, and explicit consent. A hidden website field rejects simple automated submissions.
3. The API creates a tenant-scoped conversation and a seven-day web-chat session.
4. The raw session token is returned once and stored by the widget in the current tab's `sessionStorage`.
5. Further reads and messages require the session ID plus `X-RentStage-Chat-Token`.
6. The widget polls the session every four seconds while open and renders only inbound customer messages and outbound messages already marked `SENT`, `DELIVERED`, or `READ`.

## Operator flow

Each inbound web message appears in the existing authenticated assistant inbox with channel `WEB_CHAT`. RentStage prepares a `DRAFT`, but the draft is private to the operator interface. An authorized user can edit it and press **Publish to web chat**. Publication records the authenticated sender and makes that response visible only to the matching public session.

New inbound context supersedes older unsent drafts so the operator reviews one current response. No assistant action accepts a quote or reserves inventory automatically.

## Data and security controls

- Session tokens contain 256 random bits and are encoded as 43-character base64url values.
- PostgreSQL stores only the SHA-256 token hash.
- Invalid tokens return the same public outcome as unavailable sessions.
- Public chat responses use `Cache-Control: no-store`, `Pragma: no-cache`, `Referrer-Policy: no-referrer`, and `X-Content-Type-Options: nosniff`.
- Client message UUIDs make retries idempotent.
- Messages are limited to 2,000 Unicode characters and 60 inbound messages per rolling hour.
- Session access is bound to the tenant slug, session ID, token hash, active tenant, published catalog, and enabled web-chat setting.
- The browser proxy forwards the dedicated chat-token header without exposing the internal API origin.
- Raw tokens must not appear in logs, audit metadata, assistant messages, database columns, analytics events, or URLs.

## Deliberate v0.19.0 boundaries

- Delivery uses short polling, not WebSockets or Server-Sent Events.
- Browser-tab continuity uses `sessionStorage`; it does not synchronize across devices or tabs.
- No attachments, typing indicators, read receipts, push notifications, transcript email, or automatic customer linking are provided.
- Instagram and Messenger delivery are not enabled.
- Meta cloud outbound remains blocked; the existing Meta local harness remains isolated.
- No production infrastructure or application deployment is performed by this release.
