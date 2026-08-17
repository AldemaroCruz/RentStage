export type Tenant = {
  id: string;
  name: string;
  slug: string;
  legal_name?: string;
  email?: string;
  phone?: string;
  logo_url?: string;
  address?: string;
  country_code: string;
  timezone: string;
  currency: string;
  status: string;
};

export type Category = {
  id: string;
  name: string;
  description: string;
  resource_count: number;
  created_at: string;
  updated_at: string;
};

export type Resource = {
  id: string;
  category_id?: string;
  category_name?: string;
  resource_type: "EQUIPMENT" | "SPACE" | "SERVICE";
  name: string;
  description: string;
  sku?: string;
  base_price: number;
  pricing_unit: "HOUR" | "DAY" | "EVENT" | "FIXED";
  deposit_amount: number;
  track_individual_assets: boolean;
  active: boolean;
  metadata: Record<string, unknown>;
  asset_count: number;
  available_asset_count: number;
  attention_asset_count: number;
  created_at: string;
  updated_at: string;
};

export type PackagePricingMode = "SUM_ITEMS" | "FIXED";

export type RentalPackageSummary = {
  id: string;
  name: string;
  slug: string;
  description: string;
  guest_capacity?: number;
  pricing_mode: PackagePricingMode;
  fixed_price?: number;
  image_url?: string;
  public_visible: boolean;
  public_featured: boolean;
  public_sort_order: number;
  calculated_price: number;
  effective_price: number;
  discount_value: number;
  surcharge_value: number;
  item_count: number;
  total_quantity: number;
  unavailable_item_count: number;
  ready: boolean;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type RentalPackageItem = {
  id: string;
  resource_id: string;
  resource_name: string;
  resource_type: "EQUIPMENT" | "SPACE" | "SERVICE";
  pricing_unit: "HOUR" | "DAY" | "EVENT" | "FIXED";
  resource_active: boolean;
  description: string;
  quantity: number;
  base_price: number;
  unit_price_override?: number;
  unit_price: number;
  line_total: number;
  asset_count: number;
  available_asset_count: number;
  attention_asset_count: number;
  sort_order: number;
  created_at: string;
  updated_at: string;
};

export type PackageQuoteTemplateItem = {
  resource_id: string;
  resource_name: string;
  description: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  line_total: number;
};

export type PackageQuoteTemplate = {
  package_id: string;
  package_name: string;
  package_quantity: number;
  pricing_mode: PackagePricingMode;
  calculated_price: number;
  effective_price: number;
  discount_amount: number;
  extra_charges: number;
  items: PackageQuoteTemplateItem[];
};

export type RentalPackage = RentalPackageSummary & {
  items: RentalPackageItem[];
  quote_template: PackageQuoteTemplate;
};

export type PackageAvailabilityResult = {
  package_id: string;
  package_name: string;
  package_quantity: number;
  start_at: string;
  end_at: string;
  available: boolean;
  items: AvailabilityItem[];
};

export type AssetStatus =
  | "AVAILABLE"
  | "MAINTENANCE"
  | "DAMAGED"
  | "LOST"
  | "RETIRED";

export type Asset = {
  id: string;
  resource_id: string;
  resource_name: string;
  asset_code: string;
  serial_number?: string;
  physical_status: AssetStatus;
  purchase_date?: string;
  purchase_price?: number;
  notes: string;
  created_at: string;
  updated_at: string;
};

export type DashboardData = {
  metrics: {
    active_resources: number;
    total_assets: number;
    available_assets: number;
    attention_assets: number;
    active_reservations: number;
    inventory_investment: number;
    today_departures: number;
    today_returns: number;
    overdue_returns: number;
    active_value: number;
  };
  categories: Array<{
    id: string;
    name: string;
    resource_count: number;
    asset_count: number;
  }>;
  recent_resources: Array<{
    id: string;
    name: string;
    category_name?: string;
    base_price: number;
    pricing_unit: string;
    asset_count: number;
    available_asset_count: number;
  }>;
};

export type AuditEvent = {
  id: string;
  actor_type: string;
  actor_id: string;
  actor_name?: string;
  actor_email?: string;
  action: string;
  entity_type: string;
  entity_id?: string;
  metadata: Record<string, unknown>;
  request_id?: string;
  created_at: string;
};

export type CustomerSource = "WEB" | "WHATSAPP" | "MANUAL" | "IMPORT";

export type Customer = {
  id: string;
  first_name: string;
  last_name: string;
  display_name: string;
  phone?: string;
  email?: string;
  company_name?: string;
  tax_id: string;
  tax_registration_number: string;
  billing_address: string;
  document_type_code: string;
  trade_name: string;
  economic_activity: string;
  economic_activity_code: string;
  department_code: string;
  municipality_code: string;
  district_code: string;
  preferred_language: "es" | "en";
  source: CustomerSource;
  notes: string;
  quote_count: number;
  accepted_quote_count: number;
  accepted_quote_revenue: number;
  last_quote_at?: string;
  created_at: string;
  updated_at: string;
};

export type QuoteStatus =
  | "DRAFT"
  | "SENT"
  | "ACCEPTED"
  | "REJECTED"
  | "EXPIRED"
  | "CANCELLED";

export type QuoteItem = {
  id: string;
  resource_id: string;
  resource_name: string;
  description: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  line_total: number;
  created_at: string;
};

export type QuoteSummary = {
  id: string;
  quote_number: number;
  customer_id: string;
  customer_name: string;
  customer_phone?: string;
  reservation_id?: string;
  reservation_number?: number;
  status: QuoteStatus;
  start_at: string;
  end_at: string;
  event_type?: string;
  event_location?: string;
  subtotal: number;
  discount_amount: number;
  extra_charges: number;
  total: number;
  item_count: number;
  expires_at?: string;
  created_at: string;
  updated_at: string;
};

export type QuotePortalStatus = "ACTIVE" | "ACCEPTED" | "REJECTED" | "REVOKED" | "EXPIRED";

export type QuotePortalSummary = {
  id: string;
  status: QuotePortalStatus;
  revision: number;
  expires_at: string;
  last_viewed_at?: string;
  view_count: number;
  decision_at?: string;
  decision_source?: "CUSTOMER" | "ADMIN" | "SYSTEM";
  response_name?: string;
  response_email?: string;
  rejection_reason?: string;
  terms_version: string;
  created_at: string;
  updated_at: string;
  public_url?: string;
};

export type QuoteDetail = QuoteSummary & {
  notes: string;
  items: QuoteItem[];
  portal?: QuotePortalSummary;
};

export type QuotePortalSettings = {
  tenant_id: string;
  enabled: boolean;
  headline: string;
  introduction: string;
  accent_color: string;
  default_validity_days: number;
  allow_rejection: boolean;
  require_response_name: boolean;
  acceptance_terms_text: string;
  acceptance_terms_version: string;
  created_at: string;
  updated_at: string;
};

export type PublicQuotePortal = {
  status: QuotePortalStatus;
  headline: string;
  introduction: string;
  accent_color: string;
  allow_rejection: boolean;
  require_response_name: boolean;
  terms_text: string;
  terms_version: string;
  expires_at: string;
  decision_at?: string;
  decision_source?: "CUSTOMER" | "ADMIN" | "SYSTEM";
  response_name?: string;
  rejection_reason?: string;
  can_accept: boolean;
  can_reject: boolean;
};

export type PublicQuotePortalItem = {
  description: string;
  resource_name: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  line_total: number;
};

export type PublicQuotePortalQuote = {
  quote_number: number;
  status: QuoteStatus;
  customer_name: string;
  start_at: string;
  end_at: string;
  event_type?: string;
  event_location?: string;
  subtotal: number;
  discount_amount: number;
  extra_charges: number;
  total: number;
  created_at: string;
  reservation_number?: number;
  items: PublicQuotePortalItem[];
};

export type PublicQuotePortalView = {
  tenant: PublicTenant;
  portal: PublicQuotePortal;
  quote: PublicQuotePortalQuote;
};

export type QuotePortalDecisionResult = {
  status: "ACCEPTED" | "REJECTED";
  quote_number: number;
  reservation_number?: number;
  decision_at: string;
  idempotent: boolean;
};

export type QuotePortalAvailabilityConflict = {
  available: false;
  items: Array<{
    resource_name: string;
    requested_quantity: number;
    can_fulfill: boolean;
  }>;
};

export type AvailabilityItem = {
  resource_id: string;
  resource_name: string;
  requested_quantity: number;
  eligible_assets: number;
  reserved_quantity: number;
  available_quantity: number;
  can_fulfill: boolean;
};

export type AvailabilityResult = {
  start_at: string;
  end_at: string;
  available: boolean;
  items: AvailabilityItem[];
};

export type ReservationSource = "QUOTE" | "MANUAL" | "WEB" | "WHATSAPP" | "AI_AGENT";

export type ReservationStatus =
  | "PENDING"
  | "CONFIRMED"
  | "PREPARING"
  | "READY"
  | "CHECKED_OUT"
  | "RETURNED"
  | "COMPLETED"
  | "CANCELLED";

export type AssignmentState = "ASSIGNED" | "CHECKED_OUT" | "RETURNED" | "RELEASED";

export type ReturnCondition = "GOOD" | "MAINTENANCE_REQUIRED" | "DAMAGED" | "LOST";

export type AssignedAsset = {
  assignment_id: string;
  asset_id: string;
  asset_code: string;
  serial_number?: string;
  physical_status: AssetStatus;
  state: AssignmentState;
  assigned_at: string;
  assigned_by: string;
  checked_out_at?: string;
  checked_out_by?: string;
  returned_at?: string;
  returned_by?: string;
  return_condition?: ReturnCondition;
  return_notes: string;
  released_at?: string;
  released_by?: string;
  release_reason: string;
};

export type ReservationItem = {
  id: string;
  resource_id: string;
  resource_name: string;
  description: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  line_total: number;
  track_individual_assets: boolean;
  assigned_quantity: number;
  missing_quantity: number;
  assignments: AssignedAsset[];
};

export type ReservationStatusHistory = {
  id: string;
  from_status?: ReservationStatus;
  to_status: ReservationStatus;
  actor_id: string;
  note: string;
  created_at: string;
};

export type ReservationActivityEvent = {
  id: string;
  event_type:
    | "ASSET_ASSIGNED"
    | "ASSET_UNASSIGNED"
    | "ASSET_CHECKED_OUT"
    | "ASSET_RETURNED"
    | "ASSIGNMENTS_RELEASED";
  asset_id?: string;
  asset_code?: string;
  resource_name?: string;
  actor_id: string;
  note: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type ReservationSummary = {
  id: string;
  reservation_number: number;
  customer_id: string;
  customer_name: string;
  customer_phone?: string;
  quote_id?: string;
  quote_number?: number;
  source: ReservationSource;
  status: ReservationStatus;
  block_start_at: string;
  block_end_at: string;
  event_start_at: string;
  event_end_at: string;
  event_type?: string;
  event_location?: string;
  subtotal: number;
  discount_amount: number;
  extra_charges: number;
  total: number;
  item_count: number;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  checked_out_at?: string;
  checked_out_by?: string;
  returned_at?: string;
  returned_by?: string;
};

export type ReservationScheduleHistory = {
  id: string;
  previous_block_start_at: string;
  previous_block_end_at: string;
  previous_event_start_at: string;
  previous_event_end_at: string;
  new_block_start_at: string;
  new_block_end_at: string;
  new_event_start_at: string;
  new_event_end_at: string;
  reason: string;
  actor_id: string;
  created_at: string;
};

export type ReservationDetail = ReservationSummary & {
  notes: string;
  checkout_notes: string;
  return_notes: string;
  warehouse_complete: boolean;
  required_asset_count: number;
  assigned_asset_count: number;
  items: ReservationItem[];
  status_history: ReservationStatusHistory[];
  activity_history: ReservationActivityEvent[];
  schedule_history: ReservationScheduleHistory[];
};

export type AssignableAsset = {
  id: string;
  resource_id: string;
  asset_code: string;
  serial_number?: string;
  physical_status: AssetStatus;
  notes: string;
};

export type WarehouseItem = {
  reservation_item_id: string;
  resource_id: string;
  resource_name: string;
  track_individual_assets: boolean;
  required_quantity: number;
  assigned_quantity: number;
  missing_quantity: number;
  assignments: AssignedAsset[];
  available_assets: AssignableAsset[];
};

export type WarehouseInventory = {
  reservation_id: string;
  status: ReservationStatus;
  can_manage_assignments: boolean;
  complete: boolean;
  required_asset_count: number;
  assigned_asset_count: number;
  items: WarehouseItem[];
};

export type CalendarReservation = {
  id: string;
  reservation_number: number;
  customer_id: string;
  customer_name: string;
  customer_phone?: string;
  source: ReservationSource;
  status: ReservationStatus;
  block_start_at: string;
  block_end_at: string;
  event_start_at: string;
  event_end_at: string;
  event_type?: string;
  event_location?: string;
  total: number;
  item_count: number;
  required_asset_count: number;
  assigned_asset_count: number;
  resource_summary: string;
  checked_out_at?: string;
  returned_at?: string;
};

export type CalendarResult = {
  from: string;
  to: string;
  timezone: string;
  items: CalendarReservation[];
};

export type OperationAlertSeverity = "CRITICAL" | "WARNING" | "INFO";

export type OperationAlert = {
  id: string;
  type:
    | "OVERDUE_RETURN"
    | "PREPARATION_NOT_STARTED"
    | "PREPARATION_INCOMPLETE"
    | "CHECKOUT_PENDING"
    | "COMPLETION_PENDING";
  severity: OperationAlertSeverity;
  reservation_id: string;
  reservation_number: number;
  customer_name: string;
  event_type?: string;
  status: ReservationStatus;
  title: string;
  message: string;
  due_at?: string;
  missing_asset_count: number;
  required_asset_count: number;
  assigned_asset_count: number;
  minutes_overdue: number;
};

export type OperationAlertsResult = {
  generated_at: string;
  counts: { critical: number; warning: number; info: number; total: number };
  items: OperationAlert[];
};

export type OperationsAgenda = {
  date: string;
  timezone: string;
  day_start: string;
  day_end: string;
  departures: CalendarReservation[];
  events: CalendarReservation[];
  returns: CalendarReservation[];
  pending_close: CalendarReservation[];
  overdue_returns: OperationAlert[];
};



export type PublicCatalogSettings = {
  tenant_id: string;
  enabled: boolean;
  headline: string;
  description: string;
  cover_image_url?: string;
  accent_color: string;
  show_prices: boolean;
  show_resources: boolean;
  quote_requests_enabled: boolean;
  contact_email?: string;
  contact_phone?: string;
  contact_address?: string;
  terms_text: string;
  terms_version: string;
  created_at: string;
  updated_at: string;
};

export type PublicTenant = {
  name: string;
  slug: string;
  logo_url?: string;
  email?: string;
  phone?: string;
  address?: string;
  currency: string;
  timezone: string;
};

export type PublicCatalogViewSettings = Omit<PublicCatalogSettings, "tenant_id" | "enabled" | "created_at" | "updated_at">;

export type AdminPublicPackage = {
  id: string;
  name: string;
  slug: string;
  description: string;
  guest_capacity?: number;
  image_url?: string;
  effective_price: number;
  item_count: number;
  total_quantity: number;
  ready: boolean;
  active: boolean;
  public_visible: boolean;
  public_featured: boolean;
  public_sort_order: number;
};

export type AdminPublicResource = {
  id: string;
  category_name?: string;
  resource_type: "EQUIPMENT" | "SPACE" | "SERVICE";
  name: string;
  description: string;
  base_price: number;
  pricing_unit: "HOUR" | "DAY" | "EVENT" | "FIXED";
  active: boolean;
  public_slug?: string;
  public_description: string;
  public_image_url?: string;
  public_visible: boolean;
  public_featured: boolean;
  public_sort_order: number;
};

export type AdminPublicCatalog = {
  settings: PublicCatalogSettings;
  tenant: PublicTenant;
  public_url: string;
  packages: AdminPublicPackage[];
  resources: AdminPublicResource[];
};

export type PublicPackageSummary = {
  name: string;
  slug: string;
  description: string;
  guest_capacity?: number;
  image_url?: string;
  effective_price?: number;
  item_count: number;
  total_quantity: number;
  featured: boolean;
};

export type PublicPackageItem = {
  resource_name: string;
  description: string;
  quantity: number;
};

export type PublicPackageDetail = PublicPackageSummary & {
  items: PublicPackageItem[];
};

export type PublicResourceItem = {
  slug: string;
  category_name?: string;
  resource_type: "EQUIPMENT" | "SPACE" | "SERVICE";
  name: string;
  description: string;
  image_url?: string;
  base_price?: number;
  pricing_unit: "HOUR" | "DAY" | "EVENT" | "FIXED";
  featured: boolean;
};

export type PublicCatalog = {
  tenant: PublicTenant;
  settings: PublicCatalogViewSettings;
  packages: PublicPackageSummary[];
  resources: PublicResourceItem[];
};

export type PublicPackageResponse = {
  tenant: PublicTenant;
  settings: PublicCatalogViewSettings;
  package: PublicPackageDetail;
};

export type PublicResourceResponse = {
  tenant: PublicTenant;
  settings: PublicCatalogViewSettings;
  resource: PublicResourceItem;
};

export type QuoteRequestSelection = {
  package_slug: string;
  quantity: number;
};

export type PublicAvailabilityItem = {
  resource_name: string;
  requested_quantity: number;
  can_fulfill: boolean;
};

export type PublicAvailabilityResult = {
  start_at: string;
  end_at: string;
  available: boolean;
  items: PublicAvailabilityItem[];
};

export type PublicQuoteRequestInput = {
  first_name: string;
  last_name: string;
  phone?: string;
  email?: string;
  company_name?: string;
  preferred_language: "es" | "en";
  event_type?: string;
  event_location?: string;
  start_at: string;
  end_at: string;
  notes: string;
  consent_accepted: boolean;
  website: string;
  selections: QuoteRequestSelection[];
};

export type QuoteRequestReceipt = {
  reference_code: string;
  status: "NEW";
  estimated_total?: number;
  availability_available: boolean;
  availability: PublicAvailabilityResult;
  created_at: string;
};

export type QuoteRequestStatus = "NEW" | "IN_REVIEW" | "CONVERTED" | "CLOSED" | "SPAM";

export type QuoteRequestSummary = {
  id: string;
  reference_code: string;
  status: QuoteRequestStatus;
  customer_name: string;
  phone?: string;
  email?: string;
  event_type?: string;
  event_location?: string;
  start_at: string;
  end_at: string;
  estimated_total: number;
  currency: string;
  availability_available: boolean;
  package_count: number;
  converted_quote_id?: string;
  handled_at?: string;
  created_at: string;
  updated_at: string;
};

export type QuoteRequestPackage = {
  id: string;
  package_id?: string;
  package_name: string;
  package_slug: string;
  quantity: number;
  unit_price: number;
  line_total: number;
  template: PackageQuoteTemplate;
  created_at: string;
};

export type QuoteRequestItem = {
  id: string;
  resource_id: string;
  resource_name: string;
  description: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  line_total: number;
  created_at: string;
};

export type QuoteRequestDetail = QuoteRequestSummary & {
  first_name: string;
  last_name: string;
  company_name?: string;
  preferred_language: "es" | "en";
  notes: string;
  estimated_subtotal: number;
  estimated_discount_amount: number;
  estimated_extra_charges: number;
  availability: AvailabilityResult;
  terms_text: string;
  terms_version: string;
  consent_accepted: boolean;
  converted_customer_id?: string;
  packages: QuoteRequestPackage[];
  items: QuoteRequestItem[];
};

export type QuoteRequestList = {
  items: QuoteRequestSummary[];
  counts: Partial<Record<QuoteRequestStatus, number>>;
};

export type QuoteRequestConversion = {
  request_id: string;
  reference_code: string;
  customer_id: string;
  quote_id: string;
  quote_number: number;
};


export type BillingSettings = {
  tenant_id: string;
  enabled: boolean;
  legal_name: string;
  trade_name: string;
  tax_id: string;
  tax_registration_number: string;
  economic_activity: string;
  economic_activity_code: string;
  fiscal_address: string;
  department: string;
  municipality: string;
  district: string;
  department_code: string;
  municipality_code: string;
  district_code: string;
  email: string;
  phone: string;
  prices_include_tax: boolean;
  default_tax_rate: number;
  default_payment_terms_days: number;
  invoice_prefix: string;
  next_invoice_number: number;
  fiscal_profile_complete: boolean;
  fiscal_profile_missing_fields: string[];
  created_at: string;
  updated_at: string;
};

export type TaxRule = {
  id: string;
  code: string;
  name: string;
  category: "TAXABLE" | "EXEMPT" | "NON_TAXABLE";
  rate: number;
  active: boolean;
  is_default: boolean;
  valid_from: string;
  valid_until?: string;
  created_at: string;
  updated_at: string;
};

export type InvoiceStatus = "DRAFT" | "ISSUED" | "PARTIALLY_PAID" | "PAID" | "VOID";
export type InvoiceDisplayStatus = InvoiceStatus | "OVERDUE";

export type InvoiceItem = {
  id: string;
  resource_id?: string;
  tax_rule_id?: string;
  description: string;
  quantity: number;
  unit_price: number;
  discount_amount: number;
  gross_amount: number;
  net_amount: number;
  tax_code: string;
  tax_category: "TAXABLE" | "EXEMPT" | "NON_TAXABLE";
  tax_rate: number;
  tax_amount: number;
  line_total: number;
  sort_order: number;
  dte_item_type?: number;
  dte_unit_code?: number;
  dte_product_code?: string;
};

export type InvoiceSummary = {
  id: string;
  invoice_number?: number;
  invoice_prefix: string;
  display_number: string;
  customer_id: string;
  customer_name: string;
  quote_id?: string;
  quote_number?: number;
  reservation_id?: string;
  reservation_number?: number;
  source_type: "MANUAL" | "QUOTE" | "RESERVATION";
  status: InvoiceStatus;
  display_status: InvoiceDisplayStatus;
  issue_date: string;
  due_date: string;
  currency: string;
  prices_include_tax: boolean;
  taxable_amount: number;
  exempt_amount: number;
  non_taxable_amount: number;
  tax_amount: number;
  total_amount: number;
  paid_amount: number;
  balance_due: number;
  fiscal_status: string;
  item_count: number;
  issued_at?: string;
  voided_at?: string;
  created_at: string;
  updated_at: string;
};

export type InvoiceEvent = {
  id: string;
  event_type: string;
  actor_id: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type PaymentAllocation = {
  id: string;
  payment_id?: string;
  payment_number?: number;
  payment_display_number?: string;
  invoice_id: string;
  invoice_number?: number;
  invoice_prefix: string;
  display_number: string;
  amount: number;
};

export type InvoiceDetail = InvoiceSummary & {
  customer_tax_id: string;
  customer_email: string;
  customer_phone: string;
  customer_address: string;
  seller_legal_name: string;
  seller_trade_name: string;
  seller_tax_id: string;
  seller_registration_number: string;
  seller_economic_activity: string;
  seller_economic_activity_code: string;
  seller_address: string;
  seller_email: string;
  seller_phone: string;
  notes: string;
  terms: string;
  void_reason: string;
  items: InvoiceItem[];
  events: InvoiceEvent[];
  allocations: PaymentAllocation[];
};

export type PaymentStatus = "CONFIRMED" | "VOIDED";
export type PaymentMethod = "CASH" | "BANK_TRANSFER" | "CARD" | "CHECK" | "OTHER";

export type PaymentSummary = {
  id: string;
  payment_number: number;
  display_number: string;
  customer_id: string;
  customer_name: string;
  status: PaymentStatus;
  amount: number;
  currency: string;
  method: PaymentMethod;
  reference: string;
  received_at: string;
  allocation_count: number;
  voided_at?: string;
  created_at: string;
  updated_at: string;
};

export type PaymentDetail = PaymentSummary & {
  notes: string;
  void_reason: string;
  allocations: PaymentAllocation[];
};

export type SecurityDepositStatus =
  | "PENDING"
  | "RECEIVED"
  | "PARTIALLY_SETTLED"
  | "RETURNED"
  | "RETAINED"
  | "SETTLED";

export type SecurityDeposit = {
  id: string;
  deposit_number: number;
  display_number: string;
  reservation_id: string;
  reservation_number: number;
  customer_id: string;
  customer_name: string;
  status: SecurityDepositStatus;
  amount: number;
  returned_amount: number;
  retained_amount: number;
  balance_amount: number;
  currency: string;
  method: PaymentMethod;
  reference: string;
  notes: string;
  received_at?: string;
  settled_at?: string;
  settlement_reason: string;
  created_at: string;
  updated_at: string;
};

export type BillingDashboard = {
  generated_at: string;
  currency: string;
  settings: BillingSettings;
  metrics: {
    issued_total: number;
    collected_total: number;
    outstanding_total: number;
    overdue_total: number;
    tax_output_total: number;
    deposits_held_total: number;
    draft_count: number;
    open_invoice_count: number;
    overdue_count: number;
    paid_count: number;
  };
  recent_invoices: InvoiceSummary[];
  recent_payments: PaymentSummary[];
  monthly_billing: Array<{ month: string; amount: number }>;
  monthly_payments: Array<{ month: string; amount: number }>;
};

export type Role = "OWNER" | "ADMIN" | "MANAGER" | "STAFF";

export type Permission =
  | "tenant.read"
  | "tenant.manage"
  | "team.manage"
  | "audit.read"
  | "catalog.read"
  | "catalog.manage"
  | "package.read"
  | "package.manage"
  | "public_catalog.read"
  | "public_catalog.manage"
  | "quote_request.read"
  | "quote_request.manage"
  | "inventory.read"
  | "inventory.manage"
  | "customer.read"
  | "customer.manage"
  | "quote.read"
  | "quote.manage"
  | "billing.read"
  | "billing.manage"
  | "payment.read"
  | "payment.manage"
  | "fiscal.read"
  | "fiscal.manage"
  | "reservation.read"
  | "reservation.manage"
  | "warehouse.operate"
  | "operations.read";

export type AuthUser = {
  id: string;
  identity_uid: string;
  email: string;
  display_name: string;
  avatar_url?: string;
  email_verified: boolean;
  status: string;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
};

export type Workspace = {
  tenant_id: string;
  name: string;
  slug: string;
  logo_url?: string;
  country_code: string;
  timezone: string;
  currency: string;
  tenant_status: string;
  role: Role;
  membership_status: string;
};

export type AuthMe = {
  user: AuthUser;
  workspaces: Workspace[];
  active_workspace?: Workspace;
  permissions: Permission[];
};

export type TeamMember = {
  user_id: string;
  email: string;
  display_name: string;
  avatar_url?: string;
  role: Role;
  status: "ACTIVE" | "SUSPENDED" | "REMOVED" | "INVITED";
  email_verified: boolean;
  joined_at?: string;
  last_login_at?: string;
  created_at: string;
};

export type TeamInvitation = {
  id: string;
  tenant_id: string;
  tenant_name: string;
  email: string;
  role: Exclude<Role, "OWNER">;
  status: "PENDING" | "ACCEPTED" | "REVOKED" | "EXPIRED";
  expires_at: string;
  invited_by_id: string;
  invited_by: string;
  accepted_by?: string;
  accepted_at?: string;
  created_at: string;
  updated_at: string;
  accept_url?: string;
};

export type TeamResult = {
  members: TeamMember[];
  invitations: TeamInvitation[];
};


export type DTEProviderMode = "MOCK" | "MH_HTTP";
export type DTEEnvironment = "TEST" | "PRODUCTION";
export type DTEStatus =
  | "READY_TO_SIGN"
  | "SUBMITTING"
  | "ACCEPTED"
  | "REJECTED"
  | "RETRY_REQUIRED"
  | "INVALIDATION_PENDING"
  | "INVALIDATED"
  | "CANCELLED";

export type DTESettings = {
  tenant_id: string;
  enabled: boolean;
  provider_mode: DTEProviderMode;
  environment: DTEEnvironment;
  default_document_type: "01" | "03";
  schema_version: number;
  establishment_type: string;
  establishment_code: string;
  point_of_sale_code: string;
  auth_url: string;
  signer_url: string;
  reception_url: string;
  invalidation_url: string;
  query_url: string;
  user_secret_ref: string;
  password_secret_ref: string;
  signing_password_secret_ref: string;
  auto_submit_on_issue: boolean;
  max_attempts: number;
  retry_base_seconds: number;
  next_control_number: number;
  configuration_ready: boolean;
  production_safety_ready: boolean;
  missing_configuration: string[];
  created_at: string;
  updated_at: string;
};

export type DTEEvent = {
  id: string;
  event_type: string;
  actor_id?: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type DTEDocumentSummary = {
  id: string;
  invoice_id: string;
  invoice_number?: number;
  invoice_prefix: string;
  invoice_display_number: string;
  customer_name: string;
  document_type: "01" | "03";
  document_type_label: string;
  schema_version: number;
  provider_mode: DTEProviderMode;
  environment: DTEEnvironment;
  status: DTEStatus;
  generation_code: string;
  control_number: string;
  receipt_seal: string;
  provider_status: string;
  error_code: string;
  error_message: string;
  attempt_count: number;
  next_attempt_at?: string;
  submitted_at?: string;
  accepted_at?: string;
  rejected_at?: string;
  invalidated_at?: string;
  created_at: string;
  updated_at: string;
};

export type DTEDocumentDetail = DTEDocumentSummary & {
  idempotency_key: string;
  payload: Record<string, unknown>;
  signed_document: string;
  provider_request: Record<string, unknown>;
  provider_response: Record<string, unknown>;
  invalidation_request: Record<string, unknown>;
  invalidation_response: Record<string, unknown>;
  invalidation_reason: string;
  events: DTEEvent[];
};
