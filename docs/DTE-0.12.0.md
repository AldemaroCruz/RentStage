# RentStage v0.12.0 — El Salvador DTE integration design

## Product boundary

RentStage keeps four records separate:

```text
Quote         commercial proposal
Reservation   inventory/operation commitment
Invoice       internal receivable and tax snapshot
DTE           fiscal document lifecycle and provider evidence
```

A DTE can be prepared only from an issued, unpaid-or-paid internal invoice whose fiscal profile was complete when issued. Preparing a DTE never rebuilds data from the current customer or tenant profile; it uses invoice snapshots.

## Providers

### MOCK

```text
Environment  TEST only
Network      none
Signature    deterministic SHA-256 simulation
Seal         MOCK-...
Fiscal value none
```

MOCK validates RentStage transitions, payload persistence, numbering, audit, retries, and invalidation without contacting an external service.

### MH_HTTP

MH_HTTP is a configurable adapter for the authentication, signer, reception, and invalidation endpoints supplied during official onboarding.

```text
RentStage database
  stores endpoint URLs and env:// references

Runtime environment
  contains actual credentials/signing password

MH_HTTP provider
  resolves secrets → authenticates → signs → transmits → persists redacted evidence
```

It is deliberately not hardcoded to undocumented or copied community endpoints. The adapter must be validated against the current official contract assigned to the taxpayer before use.

## State machine

```text
READY_TO_SIGN
      │ submit
      ▼
SUBMITTING
      ├── accepted ───────────────► ACCEPTED
      ├── retryable failure ──────► RETRY_REQUIRED ── retry ──┐
      └── terminal rejection ────► REJECTED                    │
                                                             └─► SUBMITTING

READY_TO_SIGN / RETRY_REQUIRED
      └── cancel ────────────────► CANCELLED

ACCEPTED
      │ invalidate
      ▼
INVALIDATION_PENDING
      ├── accepted ───────────────► INVALIDATED
      └── failed ─────────────────► ACCEPTED
```

Invoice fiscal status follows the DTE:

```text
prepare                  READY_FOR_DTE remains
submission starts        SUBMITTED
accepted                 ACCEPTED
terminal rejection       REJECTED
retryable failure        READY_FOR_DTE
cancelled preparation    READY_FOR_DTE
successful invalidation  VOIDED
```

## Numbering and idempotency

Each tenant has an independent `next_control_number` protected by `FOR UPDATE`.

```text
DTE-01-M001P001-000000000000001
```

Preparation consumes the number transactionally. Cancelled, rejected, and invalidated numbers are not reused.

Every document also receives:

```text
generation_code  random UUID
idempotency_key  SHA-256(tenant + invoice + generation code)
```

The same immutable generation/control identity is reused for a manual retry.

## Immutable evidence

`dte_documents` persists:

```text
payload
signed_document
provider_request (redacted)
provider_response
receipt_seal
error/status fields
invalidation_request
invalidation_response
attempt counters and timestamps
```

`dte_events` keeps the append-only operational timeline. Global RentStage audit records also identify the authenticated actor.

Passwords, tokens, and raw private-key material are not persisted. Request evidence recursively redacts keys containing password, token, or secret.

## Endpoint hardening

Tenant-configurable MH_HTTP targets are restricted:

```text
HTTP/HTTPS only
HTTPS mandatory in PRODUCTION
no URL userinfo credentials
no localhost
no loopback/private/link-local/metadata/CGNAT/benchmark IPs
DNS results checked before connection
redirect targets revalidated
maximum three redirects
TLS 1.2 minimum
response body limited to 4 MiB
request timeout 30 seconds
```

This is defense in depth against SSRF and accidental internal-network access. It does not replace official endpoint allowlisting during production deployment.

## DTE types in v0.12

```text
01  Factura
03  Comprobante de crédito fiscal
```

`AUTO` selects CCF only when the invoice snapshot contains NIT, NRC, and economic-activity data for the receiver; otherwise it uses the tenant default.

The local payload builder includes issuer, receiver, line, summary, IVA, payment-condition, extension, and appendix sections. It is a foundation for official certification, not a claim that every current DGII schema/catalog rule has been independently certified.

## Public legal/operational facts reflected by the design

The official process requires the taxpayer to be authorized as an electronic issuer. It distinguishes test and production environments; the DTE must be generated and electronically signed; a legible version is also produced; and the receipt seal gives the document fiscal validity. The official materials also describe public DTE consultation and contingency handling.

RentStage therefore treats `MOCK-...` as non-fiscal and reserves production activation until official onboarding is complete.

## Deliberate v0.12 limitations

- No automatic background worker; submission and retry are explicit administrative actions.
- `next_attempt_at` is advisory for the UI and future worker scheduling.
- No automatic status query/reconciliation despite the reserved `query_url` field.
- No certified legible DTE PDF or public-query QR.
- No contingency event submission.
- No credit note, debit note, export document, withholding document, or excluded-subject invoice.
- No direct Secret Manager SDK; Cloud Run should inject Secret Manager values into the referenced environment variables.
- No independent live DGII certification was possible in the packaging environment.

These boundaries should be closed with a real authorized taxpayer in the official test environment before production.
