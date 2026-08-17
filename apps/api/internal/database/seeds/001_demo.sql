INSERT INTO tenants (id, name, slug, legal_name, email, phone, country_code, timezone, currency)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'AudioPro Demo',
  'audiopro-demo',
  'AudioPro Demo, S.A. de C.V.',
  'hello@audiopro.demo',
  '+503 7000-0000',
  'SV',
  'America/El_Salvador',
  'USD'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO categories (id, tenant_id, name, description) VALUES
  ('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Speakers', 'Main and monitor loudspeakers'),
  ('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Subwoofers', 'Low-frequency reinforcement'),
  ('10000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Mixers', 'Analog and digital mixing consoles'),
  ('10000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'Microphones', 'Wired and wireless microphones'),
  ('10000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'Accessories', 'Stands, cabling, DI boxes, and accessories')
ON CONFLICT (id) DO NOTHING;

INSERT INTO resources (
  id, tenant_id, category_id, resource_type, name, description, sku,
  base_price, pricing_unit, deposit_amount, metadata
) VALUES
  (
    '20000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    'EQUIPMENT', 'JBL PRX815W', '15-inch powered loudspeaker', 'JBL-PRX815W',
    40.00, 'DAY', 100.00,
    '{"brand":"JBL","model":"PRX815W","power_watts":1500,"speaker_size_inches":15}'::jsonb
  ),
  (
    '20000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000002',
    'EQUIPMENT', 'QSC KS118', '18-inch powered subwoofer', 'QSC-KS118',
    65.00, 'DAY', 150.00,
    '{"brand":"QSC","model":"KS118","speaker_size_inches":18}'::jsonb
  ),
  (
    '20000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000003',
    'EQUIPMENT', 'Behringer X32 Compact', '40-input digital mixing console', 'BEH-X32C',
    85.00, 'DAY', 250.00,
    '{"brand":"Behringer","model":"X32 Compact","inputs":40}'::jsonb
  ),
  (
    '20000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000004',
    'EQUIPMENT', 'Shure SM58', 'Dynamic vocal microphone', 'SHU-SM58',
    8.00, 'DAY', 25.00,
    '{"brand":"Shure","model":"SM58","pattern":"cardioid"}'::jsonb
  )
ON CONFLICT (id) DO NOTHING;

INSERT INTO resources (
  id, tenant_id, category_id, resource_type, name, description, sku,
  base_price, pricing_unit, deposit_amount, track_individual_assets, metadata
)
SELECT
  '20000000-0000-0000-0000-000000000005',
  '00000000-0000-0000-0000-000000000001',
  category.id,
  'EQUIPMENT', 'Kit de cableado básico',
  'Cableado de señal y energía para un montaje estándar', 'KIT-CABLE-BASIC',
  20.00, 'DAY', 30.00, TRUE,
  '{"brand":"AudioPro","model":"Cable Kit Basic","contents":"XLR, power and extensions"}'::jsonb
FROM categories category
WHERE category.tenant_id = '00000000-0000-0000-0000-000000000001'
  AND category.name = 'Accessories'
  AND NOT EXISTS (
    SELECT 1 FROM resources existing
    WHERE existing.tenant_id = category.tenant_id
      AND existing.sku = 'KIT-CABLE-BASIC'
  )
ON CONFLICT DO NOTHING;

INSERT INTO assets (
  id, tenant_id, resource_id, asset_code, serial_number, physical_status,
  purchase_date, purchase_price, notes
) VALUES
  ('30000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'SPK-JBL-001', 'PRX815-DEMO-001', 'AVAILABLE', '2025-01-10', 850.00, ''),
  ('30000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'SPK-JBL-002', 'PRX815-DEMO-002', 'AVAILABLE', '2025-01-10', 850.00, ''),
  ('30000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'SPK-JBL-003', 'PRX815-DEMO-003', 'MAINTENANCE', '2025-02-11', 850.00, 'Inspect power connector'),
  ('30000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'SPK-JBL-004', 'PRX815-DEMO-004', 'AVAILABLE', '2025-02-11', 850.00, ''),
  ('30000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002', 'SUB-QSC-001', 'KS118-DEMO-001', 'AVAILABLE', '2025-03-05', 1999.00, ''),
  ('30000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000002', 'SUB-QSC-002', 'KS118-DEMO-002', 'AVAILABLE', '2025-03-05', 1999.00, ''),
  ('30000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000003', 'MIX-X32-001', 'X32C-DEMO-001', 'AVAILABLE', '2024-09-18', 1699.00, ''),
  ('30000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000004', 'MIC-SM58-001', 'SM58-DEMO-001', 'AVAILABLE', '2024-04-22', 99.00, ''),
  ('30000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000004', 'MIC-SM58-002', 'SM58-DEMO-002', 'AVAILABLE', '2024-04-22', 99.00, ''),
  ('30000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000004', 'MIC-SM58-003', 'SM58-DEMO-003', 'AVAILABLE', '2024-04-22', 99.00, ''),
  ('30000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000004', 'MIC-SM58-004', 'SM58-DEMO-004', 'AVAILABLE', '2024-04-22', 99.00, ''),
  ('30000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000004', 'MIC-SM58-005', 'SM58-DEMO-005', 'DAMAGED', '2024-06-13', 99.00, 'Capsule replacement required'),
  ('30000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000004', 'MIC-SM58-006', 'SM58-DEMO-006', 'AVAILABLE', '2024-06-13', 99.00, '')
ON CONFLICT (id) DO NOTHING;

INSERT INTO assets (
  id, tenant_id, resource_id, asset_code, serial_number, physical_status,
  purchase_date, purchase_price, notes
)
SELECT
  seeded.id::uuid,
  '00000000-0000-0000-0000-000000000001'::uuid,
  resource.id,
  seeded.asset_code,
  seeded.serial_number,
  'AVAILABLE',
  DATE '2025-04-01',
  180.00,
  ''
FROM (VALUES
  ('30000000-0000-0000-0000-000000000014', 'CBL-KIT-001', 'CBL-DEMO-001'),
  ('30000000-0000-0000-0000-000000000015', 'CBL-KIT-002', 'CBL-DEMO-002'),
  ('30000000-0000-0000-0000-000000000016', 'CBL-KIT-003', 'CBL-DEMO-003')
) AS seeded(id, asset_code, serial_number)
JOIN resources resource
  ON resource.tenant_id = '00000000-0000-0000-0000-000000000001'
 AND resource.sku = 'KIT-CABLE-BASIC'
ON CONFLICT DO NOTHING;

INSERT INTO packages (
  id, tenant_id, name, slug, description, guest_capacity,
  pricing_mode, fixed_price, image_url, active
)
SELECT
  '70000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'Paquete Fiesta 100 personas',
  'paquete-fiesta-100-personas',
  'Sistema completo para una fiesta de hasta 100 personas con refuerzo de bajos, mezcla, micrófonos y cableado.',
  100, 'FIXED', 299.00, NULL, TRUE
WHERE NOT EXISTS (
  SELECT 1 FROM packages existing
  WHERE existing.tenant_id = '00000000-0000-0000-0000-000000000001'
    AND existing.slug = 'paquete-fiesta-100-personas'
)
ON CONFLICT DO NOTHING;

INSERT INTO package_items (
  id, tenant_id, package_id, resource_id, description,
  quantity, unit_price_override, sort_order
)
SELECT
  seeded.id::uuid,
  package.tenant_id,
  package.id,
  resource.id,
  seeded.description,
  seeded.quantity,
  NULL,
  seeded.sort_order
FROM (VALUES
  ('71000000-0000-0000-0000-000000000001', 'JBL-PRX815W', 'Sistema principal JBL PRX815W', 2, 0),
  ('71000000-0000-0000-0000-000000000002', 'QSC-KS118', 'Refuerzo de bajos QSC KS118', 2, 1),
  ('71000000-0000-0000-0000-000000000003', 'BEH-X32C', 'Mezcladora digital Behringer X32 Compact', 1, 2),
  ('71000000-0000-0000-0000-000000000004', 'SHU-SM58', 'Micrófonos vocales Shure SM58', 2, 3),
  ('71000000-0000-0000-0000-000000000005', 'KIT-CABLE-BASIC', 'Cableado básico de señal y energía', 1, 4)
) AS seeded(id, sku, description, quantity, sort_order)
JOIN packages package
  ON package.tenant_id = '00000000-0000-0000-0000-000000000001'
 AND package.slug = 'paquete-fiesta-100-personas'
JOIN resources resource
  ON resource.tenant_id = package.tenant_id
 AND resource.sku = seeded.sku
ON CONFLICT DO NOTHING;

INSERT INTO customers (
  id, tenant_id, first_name, last_name, phone, email, company_name,
  preferred_language, source, notes
) VALUES
  (
    '40000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Carlos', 'Hernández', '+50371234567', 'carlos@example.com', NULL,
    'es', 'MANUAL', 'Cliente frecuente para fiestas privadas.'
  ),
  (
    '40000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'María', 'López', '+50372345678', 'maria@eventosmarea.example', 'Eventos Marea',
    'es', 'WHATSAPP', 'Productora de eventos corporativos y música en vivo.'
  )
ON CONFLICT (id) DO NOTHING;

INSERT INTO quotes (
  id, tenant_id, customer_id, start_at, end_at, status, event_type,
  event_location, subtotal, discount_amount, extra_charges, total, notes, expires_at
) VALUES
  (
    '50000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    '40000000-0000-0000-0000-000000000001',
    NOW() + INTERVAL '7 days',
    NOW() + INTERVAL '7 days 12 hours',
    'DRAFT',
    'Fiesta de cumpleaños',
    'San Salvador',
    218.00, 0.00, 40.00, 258.00,
    'Incluye cableado básico. Transporte sujeto a confirmación final.',
    NOW() + INTERVAL '3 days'
  ),
  (
    '50000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    '40000000-0000-0000-0000-000000000002',
    NOW() + INTERVAL '14 days',
    NOW() + INTERVAL '14 days 10 hours',
    'SENT',
    'Evento corporativo con banda',
    'Santa Tecla',
    117.00, 0.00, 25.00, 142.00,
    'Cotización enviada por el equipo comercial.',
    NOW() + INTERVAL '5 days'
  )
ON CONFLICT (id) DO NOTHING;

INSERT INTO quote_items (
  id, tenant_id, quote_id, resource_id, description, quantity,
  unit_price, discount_amount, line_total
) VALUES
  (
    '60000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    '50000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    'JBL PRX815W', 2, 40.00, 0.00, 80.00
  ),
  (
    '60000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    '50000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000002',
    'QSC KS118', 2, 65.00, 0.00, 130.00
  ),
  (
    '60000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    '50000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000004',
    'Shure SM58', 1, 8.00, 0.00, 8.00
  ),
  (
    '60000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000001',
    '50000000-0000-0000-0000-000000000002',
    '20000000-0000-0000-0000-000000000003',
    'Behringer X32 Compact', 1, 85.00, 0.00, 85.00
  ),
  (
    '60000000-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000001',
    '50000000-0000-0000-0000-000000000002',
    '20000000-0000-0000-0000-000000000004',
    'Shure SM58', 4, 8.00, 0.00, 32.00
  )
ON CONFLICT (id) DO NOTHING;

-- Identity & access demo principal. The API bootstrap links this row to the
-- Firebase Authentication emulator UID on every local startup.
INSERT INTO users (
  id, email, display_name, status, email_verified
) VALUES (
  '01000000-0000-0000-0000-000000000001',
  'owner@rentstage.local',
  'Administrador Demo',
  'ACTIVE',
  TRUE
)
ON CONFLICT (email) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  status = 'ACTIVE',
  email_verified = TRUE,
  updated_at = NOW();

INSERT INTO tenant_memberships (
  tenant_id, user_id, role, status, joined_at
)
SELECT
  '00000000-0000-0000-0000-000000000001',
  u.id,
  'OWNER',
  'ACTIVE',
  NOW()
FROM users u
WHERE LOWER(u.email) = 'owner@rentstage.local'
ON CONFLICT (tenant_id, user_id) DO UPDATE SET
  role = 'OWNER',
  status = 'ACTIVE',
  joined_at = COALESCE(tenant_memberships.joined_at, NOW()),
  updated_at = NOW();

INSERT INTO user_preferences (user_id, active_tenant_id, locale, timezone)
SELECT
  u.id,
  '00000000-0000-0000-0000-000000000001',
  'es',
  'America/El_Salvador'
FROM users u
WHERE LOWER(u.email) = 'owner@rentstage.local'
ON CONFLICT (user_id) DO UPDATE SET
  active_tenant_id = EXCLUDED.active_tenant_id,
  updated_at = NOW();

-- Public catalog demo profile and publication flags. Production environments
-- keep public publishing opt-in because this seed is disabled there. The block
-- only initializes the profile once so later admin changes survive API restarts.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM public_catalog_settings
    WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  ) THEN
    INSERT INTO public_catalog_settings (
      tenant_id, enabled, headline, description, cover_image_url, accent_color,
      show_prices, show_resources, quote_requests_enabled,
      contact_email, contact_phone, contact_address, terms_text, terms_version
    ) VALUES (
      '00000000-0000-0000-0000-000000000001',
      TRUE,
      'Audio profesional para eventos que deben sonar increíble',
      'Paquetes completos y equipo de audio para fiestas, eventos corporativos y música en vivo. Cuéntanos la fecha y preparamos una cotización a tu medida.',
      NULL,
      '#6657F7',
      TRUE,
      TRUE,
      TRUE,
      'hello@audiopro.demo',
      '+503 7000-0000',
      'San Salvador, El Salvador',
      'Al enviar esta solicitud autorizas a AudioPro Demo a contactarte para preparar una cotización. La disponibilidad mostrada es una referencia y no constituye una reserva.',
      '1.0'
    );

    UPDATE packages
    SET public_visible = TRUE,
        public_featured = TRUE,
        public_sort_order = 10
    WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
      AND slug = 'paquete-fiesta-100-personas'
      AND active = TRUE;

    UPDATE resources
    SET
      public_slug = publication.slug,
      public_description = publication.description,
      public_visible = TRUE,
      public_featured = publication.featured,
      public_sort_order = publication.sort_order
    FROM (VALUES
      ('JBL-PRX815W', 'jbl-prx815w', 'Bocina activa de 15 pulgadas para sonido principal o monitoreo con excelente cobertura.', TRUE, 10),
      ('QSC-KS118', 'qsc-ks118', 'Subwoofer activo de 18 pulgadas para reforzar graves en fiestas y presentaciones en vivo.', TRUE, 20),
      ('BEH-X32C', 'behringer-x32-compact', 'Consola digital de 40 entradas para producciones con múltiples fuentes y mezcla profesional.', FALSE, 30),
      ('SHU-SM58', 'shure-sm58', 'Micrófono dinámico vocal confiable para discursos, animación y música en vivo.', FALSE, 40),
      ('KIT-CABLE-BASIC', 'kit-cableado-basico', 'Kit de señal y energía necesario para un montaje de audio estándar.', FALSE, 50)
    ) AS publication(sku, slug, description, featured, sort_order)
    WHERE resources.tenant_id = '00000000-0000-0000-0000-000000000001'
      AND resources.sku = publication.sku
      AND resources.active = TRUE;
  END IF;
END
$$;

-- Billing & Payments Core demo defaults. These rows are operational invoice
-- settings only; they do not authorize or transmit El Salvador DTE documents.
INSERT INTO billing_settings (
  tenant_id, enabled, legal_name, trade_name, fiscal_address, email, phone,
  prices_include_tax, default_tax_rate, default_payment_terms_days,
  invoice_prefix, next_invoice_number
) VALUES (
  '00000000-0000-0000-0000-000000000001',
  TRUE,
  'AudioPro Demo, S.A. de C.V.',
  'AudioPro Demo',
  'San Salvador, El Salvador',
  'hello@audiopro.demo',
  '+503 7000-0000',
  FALSE,
  13.00,
  0,
  'INV',
  1
)
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO tax_rules (tenant_id, code, name, category, rate, active, is_default)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'IVA', 'IVA estándar', 'TAXABLE', 13.00, TRUE, TRUE),
  ('00000000-0000-0000-0000-000000000001', 'EXEMPT', 'Venta exenta', 'EXEMPT', 0.00, TRUE, FALSE),
  ('00000000-0000-0000-0000-000000000001', 'NON_TAXABLE', 'Venta no sujeta', 'NON_TAXABLE', 0.00, TRUE, FALSE)
ON CONFLICT (tenant_id, code) DO NOTHING;
