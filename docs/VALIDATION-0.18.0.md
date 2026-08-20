# RentStage v0.18.0 validation record

## Release consistency

- [ ] `VERSION`, web package, README, dashboard label, and changelog report `0.18.0`.
- [ ] repository contracts, workflow YAML, migration ordering, and sensitive-file policy pass.
- [ ] PowerShell syntax, frontend typecheck/coverage, Go formatting/tests/race/vet, Docker smoke, and vulnerability scans pass.

## Webhook and Graph contract

- [ ] GET verification returns the exact raw challenge only for the matching verify token.
- [ ] POST rejects missing, malformed, or incorrect `X-Hub-Signature-256` before calling the processor.
- [ ] the signature is calculated over the exact request bytes and the body is limited to 1 MiB.
- [ ] official-style inbound text and delivery-status envelopes are normalized.
- [ ] unsupported message types are ignored explicitly.
- [ ] Graph text send uses bearer authentication, digits-only recipient, configured version/phone-number ID, and requires a returned message ID.

## Data and workflow safety

- [ ] tenant is resolved from the enabled WhatsApp phone-number connection; a browser or webhook cannot select a tenant ID.
- [ ] repeated provider message ID creates no second inbound message or draft.
- [ ] inbound customer activity opens or extends the 24-hour service window.
- [ ] sending outside that window is blocked.
- [ ] every outbound response still requires an authenticated human action and stores provider/audit evidence.
- [ ] access token, application secret, and verification token do not enter PostgreSQL, logs, source control, or response payloads.

## Environment isolation

- [ ] `local_mock` rejects non-local `APP_ENV` and any non-loopback or non-harness Graph URL.
- [ ] staging and production contain no local Meta variables and therefore remain disabled.
- [ ] no real telephone, public tunnel, Meta account, or external Graph request is used by the smoke test.
- [ ] production apply and deploy gates remain false and no temporary production roles are granted.
