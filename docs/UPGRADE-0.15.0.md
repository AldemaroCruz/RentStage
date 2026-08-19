# Upgrade RentStage v0.14.2 → v0.15.0

v0.15.0 introduces the human-approved WhatsApp Sales Assistant. No Meta Business account, phone number, API token, Google Cloud resource, or new environment variable is required.

## Database and permissions

Migration `013_whatsapp_sales_assistant.sql` creates tenant-scoped conversations, messages, and proposals. It is additive and does not rewrite existing customer, package, quote, reservation, invoice, or audit data.

Roles receive:

- `OWNER`, `ADMIN`: `assistant.read`, `assistant.manage`
- `MANAGER`: `assistant.read`, `assistant.manage`
- `STAFF`: `assistant.read`

## Deployment

The existing pipeline runs migrations before the API starts. Deploy the normal application image; no separate infrastructure apply is necessary.

After staging deploys:

1. Sign in as the staging owner.
2. Open **WhatsApp AI** in the sidebar.
3. Inspect the seeded María López conversation or select **Simular consulta**.
4. Confirm a real package and availability result appear.
5. Select an existing customer, edit the response, and approve.
6. Open the resulting quote and confirm status `DRAFT`.
7. Verify no reservation was created and the action appears in **Auditoría**.

## Rollback

Rolling back the application hides the new route and endpoints. The additive tables may remain unused. Do not drop them if proposals have already produced quote links; application rollback does not delete business records.
