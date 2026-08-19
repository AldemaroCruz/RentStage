# Upgrade RentStage v0.15.2 → v0.15.3

v0.15.3 completes the zero-credential commercial demo from the assistant conversation to the customer's secure quote decision and adds a persistent light/dark appearance control.

## What changes

- The assistant detail API includes the linked quote, portal status, view count, decision timestamps, and reservation identifiers.
- `POST /api/v1/assistant/conversations/{conversationID}/quote/share-demo` issues or rotates the existing Quote Portal link for `DEMO` conversations.
- The assistant UI keeps the newly returned bearer URL in `sessionStorage`, shows sanitized delivery evidence, and refreshes the public decision state.
- A theme bootstrap script and button apply a stored `light` or `dark` preference before React hydration.
- The assistant smoke flow now verifies link isolation and a customer rejection that creates no reservation.

## Security boundary

The portal link contains a 256-bit bearer token in the URL fragment. The API stores only its SHA-256 digest. RentStage returns the raw URL once with `Cache-Control: no-store`; the assistant never writes it to a message, audit record, database column, query string, or server log.

The sharing endpoint requires both `assistant.manage` and `quote.manage`. It is limited to the `DEMO` channel and does not contact a telephone. Accepting or rejecting remains a direct customer action in the existing public Quote Portal. The assistant never accepts a quote or creates a reservation by itself.

## Deployment

Deploy through the existing `pipeline.yml` while staging is resumed and `STAGING_DEPLOY_ENABLED=true`. No Terraform apply, migration, secret, Meta account, phone number, environment variable, IAM change, or data reset is required.

After deployment, open **WhatsApp AI**, choose a conversation with a quote draft, select **Enviar cotización y generar portal**, and open the returned customer link in the current browser session.

## Rollback

Rolling back the application removes the assistant shortcut and theme control. Existing quote portals and customer decisions remain valid because v0.15.3 does not alter their schema or lifecycle.
