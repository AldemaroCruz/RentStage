# Operational metrics v0.16.0

RentStage v0.16.0 turns existing transactional records into a tenant-scoped commercial evidence layer. It does not install a tracking pixel, analytics cookie, external warehouse, or third-party telemetry SDK.

## Workspace

Authorized users open **Métricas** from the Operations section. The page offers 7, 30, and 90-day windows and displays:

- inquiries from public quote requests and new assistant conversations;
- quote creation, presentation, acceptance, and rejection activity;
- reservations created from accepted quotes and all operational reservations;
- first-response time for inbound assistant messages that have a later sent response;
- current quote pipeline and receivable balance snapshots;
- accepted, reserved, invoiced, and collected value during the selected window;
- a six-month value trend;
- reservation outcomes and customer-source distribution;
- human-approved messages, customer Quote Portal decisions, and audit events.

## Measurement definitions

| Measure | Definition |
| --- | --- |
| Inquiry | A public quote request or assistant conversation created in the window. |
| Quote acceptance | Accepted quotes divided by accepted plus rejected quotes created in the window. Undecided and expired quotes do not enter the denominator. |
| Quote → reservation | Reservations created in the window with a non-null source quote, divided by accepted quotes created in the same window. Manual reservations do not inflate this rate. |
| First response | Minutes from each inbound assistant message to the earliest later outbound `SENT` message in the same conversation. Rows without a response are excluded and the sample size is visible. |
| Pipeline today | Sum of current `DRAFT` and `SENT` quote totals. This is a current snapshot, not a windowed flow. |
| Outstanding today | Sum of current invoice balances in `ISSUED` or `PARTIALLY_PAID`. This is a current snapshot. |
| Collected | Confirmed payments received during the selected window. |
| Cancellation | Cancelled reservations divided by completed plus cancelled reservations created in the window. Active reservations are excluded from the outcome denominator. |

## Interpretation boundary

The funnel is an activity summary, not a closed cohort. A quote accepted today may have been created before the selected period, and later-stage records may therefore exceed an earlier-stage count. The interface explains this boundary rather than presenting a misleading sequential conversion funnel.

Dates use UTC bounds in the API and are rendered in the browser's locale. The six-month chart uses calendar months ending with the current month, independently of the selected 7/30/90-day window.

## Security and tenancy

`GET /api/v1/metrics/commercial?days=30` is protected by the existing session, selected-workspace validation, tenant middleware, and `operations.read` permission. The tenant identifier is taken only from server context and every business query filters on it.

No metric endpoint accepts a tenant identifier. No raw customer message, email, telephone, bearer token, payment reference, or fiscal payload is returned by the report.

## Deliberate limitations

- The report is calculated from PostgreSQL on request; there is no pre-aggregated warehouse in this release.
- Comparisons against a preceding period, targets, exports, and scheduled reports remain follow-on work.
- DEMO assistant activity is valid product evidence but is not proof of Meta WhatsApp delivery.
- DTE activity remains MOCK / TEST unless the taxpayer completes official onboarding and homologation.
