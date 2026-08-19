# Upgrade RentStage v0.15.0 → v0.15.1

v0.15.1 completes the interactive `DEMO` chat loop and inline customer association. It introduces no database migration, infrastructure resource, secret, environment variable, or external messaging dependency.

## Deployment

Deploy through the existing `pipeline.yml`. The API image adds three authenticated routes:

- `POST /api/v1/assistant/conversations/{conversationID}/customer`
- `POST /api/v1/assistant/conversations/{conversationID}/messages/send-demo`
- `POST /api/v1/assistant/conversations/{conversationID}/messages/receive-demo`

All use existing session, CSRF, tenant, and permission middleware. The current staging service account and Terraform state do not change.

## Post-deploy check

1. Sign in as an owner, admin, or manager and open **WhatsApp AI**.
2. Open a demo conversation with a pending proposal.
3. Create a customer from the chat or link an existing customer.
4. Approve the proposal and confirm its quote remains `DRAFT`.
5. Select **Enviar respuesta demo** and confirm the transcript says it was delivered in the simulator.
6. Select **Simular respuesta del cliente**, enter a follow-up, and confirm a new unsent assistant draft appears.
7. Deliver the follow-up and verify the audit trail contains customer-link, inbound-simulation, and demo-delivery events.

## Rollback

Application rollback removes the new UI controls and routes. Existing conversations, customer links, transcript messages, quotes, and audit records remain valid under the additive v0.15.0 schema.
