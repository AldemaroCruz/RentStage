ALTER TABLE assistant_messages
  DROP CONSTRAINT assistant_messages_status_chk;

ALTER TABLE assistant_messages
  ADD CONSTRAINT assistant_messages_status_chk
  CHECK (status IN ('RECEIVED', 'DRAFT', 'APPROVED', 'SENT', 'DELIVERED', 'READ', 'FAILED'));

CREATE TABLE assistant_message_templates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  provider text NOT NULL DEFAULT 'WHATSAPP',
  name text NOT NULL,
  language text NOT NULL DEFAULT 'es',
  category text NOT NULL CHECK (category IN ('MARKETING', 'UTILITY', 'AUTHENTICATION')),
  status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PENDING', 'APPROVED', 'REJECTED', 'PAUSED', 'DISABLED')),
  variable_count integer NOT NULL DEFAULT 0 CHECK (variable_count >= 0),
  external_template_id text,
  enabled boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, provider, name, language)
);
