# RentStage v0.15.1 validation record

## Automated contracts

- [ ] `VERSION`, `apps/web/package.json`, README, dashboard release link, and changelog report `0.15.1`.
- [ ] Go formatting, module verification, unit/race tests, vet, web typecheck/tests/build, workflow checks, sensitive-file policy, and security scans pass.
- [ ] The assistant smoke suite links a customer, creates one quote in `DRAFT`, delivers the first response in `DEMO`, receives a follow-up, creates a new `DRAFT`, and delivers the second response.
- [ ] Demo operations reject non-`DEMO` channels and tenant-scoped lookups reject foreign customer IDs.

## Product acceptance

1. Confirm the page visibly says `CANAL DEMO` and explains that delivery is simulated.
2. Confirm a pending assistant bubble is labeled `BORRADOR NO ENVIADO`.
3. Create and link a customer from the active contact. Confirm source `WHATSAPP` on the customer record.
4. Create the quote draft and select **Enviar respuesta demo**. Confirm no real phone is contacted.
5. Simulate at least two different customer follow-ups and deliver the human-reviewed responses.
6. Confirm the quote remains `DRAFT`, no reservation exists, and inventory availability is unchanged.
7. Confirm audit events identify the acting user and conversation.

## Evidence

```text
CI/CD run: ______________________________
Staging URL: ____________________________
Conversation ID: _______________________
Customer ID: ___________________________
Quote ID: ______________________________
Commit SHA: ____________________________
Validated by: __________________________
Date: __________________________________
```
