-- Calendar & Operations Center: reservation sources, manual bookings, and auditable rescheduling.

ALTER TABLE reservations
  ADD COLUMN source TEXT NOT NULL DEFAULT 'QUOTE';

UPDATE reservations
SET source = CASE WHEN quote_id IS NULL THEN 'MANUAL' ELSE 'QUOTE' END;

ALTER TABLE reservations
  ADD CONSTRAINT reservations_source_check
    CHECK (source IN ('QUOTE', 'MANUAL', 'WEB', 'WHATSAPP', 'AI_AGENT')),
  ADD CONSTRAINT reservations_quote_source_consistent
    CHECK (
      (source = 'QUOTE' AND quote_id IS NOT NULL)
      OR
      (source <> 'QUOTE' AND quote_id IS NULL)
    );

CREATE TABLE reservation_schedule_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  reservation_id UUID NOT NULL,
  previous_block_start_at TIMESTAMPTZ NOT NULL,
  previous_block_end_at TIMESTAMPTZ NOT NULL,
  previous_event_start_at TIMESTAMPTZ NOT NULL,
  previous_event_end_at TIMESTAMPTZ NOT NULL,
  new_block_start_at TIMESTAMPTZ NOT NULL,
  new_block_end_at TIMESTAMPTZ NOT NULL,
  new_event_start_at TIMESTAMPTZ NOT NULL,
  new_event_end_at TIMESTAMPTZ NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  actor_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, reservation_id)
    REFERENCES reservations(tenant_id, id)
    ON DELETE CASCADE,
  CONSTRAINT reservation_schedule_previous_block_valid
    CHECK (previous_block_end_at > previous_block_start_at),
  CONSTRAINT reservation_schedule_previous_event_valid
    CHECK (previous_event_end_at > previous_event_start_at),
  CONSTRAINT reservation_schedule_previous_event_within_block
    CHECK (
      previous_event_start_at >= previous_block_start_at
      AND previous_event_end_at <= previous_block_end_at
    ),
  CONSTRAINT reservation_schedule_new_block_valid
    CHECK (new_block_end_at > new_block_start_at),
  CONSTRAINT reservation_schedule_new_event_valid
    CHECK (new_event_end_at > new_event_start_at),
  CONSTRAINT reservation_schedule_new_event_within_block
    CHECK (
      new_event_start_at >= new_block_start_at
      AND new_event_end_at <= new_block_end_at
    )
);

CREATE INDEX idx_reservation_schedule_history_reservation
  ON reservation_schedule_history(tenant_id, reservation_id, created_at DESC, id DESC);

CREATE INDEX idx_reservations_tenant_calendar
  ON reservations(tenant_id, block_start_at, block_end_at, status);

CREATE INDEX idx_reservations_tenant_source_created
  ON reservations(tenant_id, source, created_at DESC);
