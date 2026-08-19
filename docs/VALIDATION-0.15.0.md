# RentStage v0.15.0 validation record

## Release contracts

- [ ] `VERSION`, `apps/web/package.json`, README, and changelog report `0.15.0`.
- [ ] Migration ordering passes through `013_whatsapp_sales_assistant.sql`.
- [ ] Go format, module integrity, unit/race tests, vet, web typecheck/tests/build, sensitive-file policy, and security scans pass.
- [ ] A staff role can read but cannot approve; manager/admin/owner can approve.
- [ ] Approval creates exactly one quote in `DRAFT` and no reservation.

## Product acceptance

1. Open `/assistant`; verify the channel is visibly labeled **DEMO** and **Sin Meta**.
2. Simulate the provided wedding message for a future period.
3. Confirm the recommendation names a real tenant package, configured price, and availability result.
4. Confirm the suggested response remains labeled **BORRADOR**.
5. Approve with an existing customer and verify the quote link and audit event.
6. Confirm the quote status is `DRAFT` and inventory availability is unchanged.
7. Repeat approval and confirm the API returns a conflict rather than another quote.

## Evidence

```text
CI/CD run: ______________________________
Staging URL: ____________________________
Conversation ID: ________________________
Quote ID: _______________________________
Commit SHA: _____________________________
Validated by: ___________________________
Date: __________________________________
```
