-- RentStage v0.12.1 — recover local MOCK submissions left in SUBMITTING by
-- the v0.12.0 submission-result persistence query.
--
-- Only MOCK / TEST rows are recovered automatically. A real external provider
-- could already have accepted an in-flight document, so MH_HTTP rows require
-- explicit reconciliation instead of an automatic retry.

UPDATE dte_documents
SET status = 'RETRY_REQUIRED',
    next_attempt_at = NOW(),
    error_code = 'RESULT_PERSISTENCE_RECOVERED',
    error_message = 'Recovered after the v0.12.0 local submission-result persistence failure.',
    updated_at = NOW()
WHERE status = 'SUBMITTING'
  AND provider_mode = 'MOCK'
  AND environment = 'TEST';

UPDATE invoices invoice
SET fiscal_status = 'READY_FOR_DTE',
    updated_at = NOW()
FROM dte_documents document
WHERE document.tenant_id = invoice.tenant_id
  AND document.invoice_id = invoice.id
  AND document.status = 'RETRY_REQUIRED'
  AND document.error_code = 'RESULT_PERSISTENCE_RECOVERED'
  AND invoice.fiscal_status = 'SUBMITTED';

INSERT INTO dte_events (
  tenant_id,
  dte_document_id,
  event_type,
  actor_id,
  metadata
)
SELECT document.tenant_id,
       document.id,
       'DTE_RETRY_REQUIRED',
       NULL,
       jsonb_build_object(
         'reason', 'v0.12.1_submission_result_recovery',
         'previous_status', 'SUBMITTING',
         'attempt_count', document.attempt_count
       )
FROM dte_documents document
WHERE document.status = 'RETRY_REQUIRED'
  AND document.error_code = 'RESULT_PERSISTENCE_RECOVERED'
  AND NOT EXISTS (
    SELECT 1
    FROM dte_events event
    WHERE event.tenant_id = document.tenant_id
      AND event.dte_document_id = document.id
      AND event.event_type = 'DTE_RETRY_REQUIRED'
      AND event.metadata ->> 'reason' = 'v0.12.1_submission_result_recovery'
  );
