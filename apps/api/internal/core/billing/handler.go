package billing

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	repository *Repository
	service    *Service
}

func NewHandler(repository *Repository, service *Service) *Handler {
	return &Handler{repository: repository, service: service}
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	item, err := h.repository.GetSettings(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "billing_settings_load_failed", "No fue posible cargar la configuración de facturación.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input SettingsInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.UpdateSettings(r.Context(), webutil.TenantID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "billing_settings_update_failed", "No fue posible guardar la configuración de facturación.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) TaxRules(w http.ResponseWriter, r *http.Request) {
	items, err := h.repository.ListTaxRules(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "tax_rules_load_failed", "No fue posible cargar las reglas tributarias.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	item, err := h.repository.Dashboard(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "billing_dashboard_load_failed", "No fue posible cargar el dashboard financiero.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validInvoiceStatus(status) {
		webutil.WriteValidationError(w, r, map[string]string{"status": "Estado de factura no soportado."})
		return
	}
	items, err := h.repository.ListInvoices(
		r.Context(),
		webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")),
		status,
		queryLimit(r, 100),
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "invoice_list_failed", "No fue posible cargar las facturas.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := pathUUID(w, r, "invoiceID", "invalid_invoice_id", "El ID de factura es inválido.")
	if !ok {
		return
	}
	item, err := h.repository.GetInvoice(r.Context(), webutil.TenantID(r.Context()), invoiceID)
	if errors.Is(err, ErrInvoiceNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "invoice_not_found", "Factura no encontrada.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "invoice_load_failed", "No fue posible cargar la factura.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var input CreateInvoiceInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.CreateInvoice(r.Context(), webutil.TenantID(r.Context()), input)
	h.writeInvoiceMutation(w, r, item, fields, err, http.StatusCreated)
}

func (h *Handler) UpdateInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := pathUUID(w, r, "invoiceID", "invalid_invoice_id", "El ID de factura es inválido.")
	if !ok {
		return
	}
	var input UpdateInvoiceInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.UpdateInvoice(r.Context(), webutil.TenantID(r.Context()), invoiceID, input)
	h.writeInvoiceMutation(w, r, item, fields, err, http.StatusOK)
}

func (h *Handler) IssueInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := pathUUID(w, r, "invoiceID", "invalid_invoice_id", "El ID de factura es inválido.")
	if !ok {
		return
	}
	item, err := h.service.IssueInvoice(r.Context(), webutil.TenantID(r.Context()), invoiceID)
	h.writeInvoiceMutation(w, r, item, nil, err, http.StatusOK)
}

func (h *Handler) VoidInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := pathUUID(w, r, "invoiceID", "invalid_invoice_id", "El ID de factura es inválido.")
	if !ok {
		return
	}
	var input VoidInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.VoidInvoice(r.Context(), webutil.TenantID(r.Context()), invoiceID, input)
	h.writeInvoiceMutation(w, r, item, fields, err, http.StatusOK)
}

func (h *Handler) writeInvoiceMutation(w http.ResponseWriter, r *http.Request, item InvoiceDetail, fields map[string]string, err error, status int) {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	switch {
	case errors.Is(err, ErrBillingDisabled):
		webutil.WriteError(w, r, http.StatusConflict, "billing_disabled", "La facturación está deshabilitada para esta empresa.")
	case errors.Is(err, ErrInvoiceNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "invoice_not_found", "Factura no encontrada.")
	case errors.Is(err, ErrCustomerNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "invoice_customer_not_found", "El cliente seleccionado no existe.")
	case errors.Is(err, ErrSourceNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "invoice_source_not_found", "La cotización o reserva no existe, no está aceptada o fue cancelada.")
	case errors.Is(err, ErrSourceConflict):
		webutil.WriteError(w, r, http.StatusConflict, "invoice_source_conflict", "Ya existe una factura activa para este origen.")
	case errors.Is(err, ErrTaxRuleNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "tax_rule_not_found", "La regla tributaria seleccionada no existe.")
	case errors.Is(err, ErrCurrencyMismatch):
		webutil.WriteError(w, r, http.StatusConflict, "billing_currency_mismatch", "La moneda debe coincidir con la moneda configurada para la empresa.")
	case errors.Is(err, ErrInvoiceImmutable):
		webutil.WriteError(w, r, http.StatusConflict, "invoice_immutable", "Solo las facturas en borrador pueden editarse.")
	case errors.Is(err, ErrInvoiceState):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_invoice_state", "La factura no admite esta operación en su estado actual.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "invoice_mutation_failed", "No fue posible guardar la factura.")
	default:
		webutil.WriteJSON(w, status, item)
	}
}

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && status != "CONFIRMED" && status != "VOIDED" {
		webutil.WriteValidationError(w, r, map[string]string{"status": "Estado de pago no soportado."})
		return
	}
	items, err := h.repository.ListPayments(
		r.Context(), webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")), status, queryLimit(r, 100),
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "payment_list_failed", "No fue posible cargar los pagos.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	paymentID, ok := pathUUID(w, r, "paymentID", "invalid_payment_id", "El ID de pago es inválido.")
	if !ok {
		return
	}
	item, err := h.repository.GetPayment(r.Context(), webutil.TenantID(r.Context()), paymentID)
	if errors.Is(err, ErrPaymentNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "payment_not_found", "Pago no encontrado.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "payment_load_failed", "No fue posible cargar el pago.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var input CreatePaymentInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.CreatePayment(r.Context(), webutil.TenantID(r.Context()), input)
	h.writePaymentMutation(w, r, item, fields, err, http.StatusCreated)
}

func (h *Handler) VoidPayment(w http.ResponseWriter, r *http.Request) {
	paymentID, ok := pathUUID(w, r, "paymentID", "invalid_payment_id", "El ID de pago es inválido.")
	if !ok {
		return
	}
	var input VoidInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.VoidPayment(r.Context(), webutil.TenantID(r.Context()), paymentID, input)
	h.writePaymentMutation(w, r, item, fields, err, http.StatusOK)
}

func (h *Handler) writePaymentMutation(w http.ResponseWriter, r *http.Request, item PaymentDetail, fields map[string]string, err error, status int) {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	switch {
	case errors.Is(err, ErrPaymentNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "payment_not_found", "Pago no encontrado.")
	case errors.Is(err, ErrInvoiceNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "payment_invoice_not_found", "Una de las facturas no existe.")
	case errors.Is(err, ErrInvoiceState):
		webutil.WriteError(w, r, http.StatusConflict, "payment_invoice_state", "Una de las facturas no está abierta para recibir pagos.")
	case errors.Is(err, ErrCustomerMismatch):
		webutil.WriteError(w, r, http.StatusConflict, "payment_customer_mismatch", "Todas las facturas deben pertenecer al cliente del pago.")
	case errors.Is(err, ErrCurrencyMismatch):
		webutil.WriteError(w, r, http.StatusConflict, "payment_currency_mismatch", "Todas las facturas deben usar la misma moneda del pago.")
	case errors.Is(err, ErrAllocationConflict):
		webutil.WriteError(w, r, http.StatusConflict, "payment_allocation_conflict", "Una asignación supera el saldo pendiente de la factura.")
	case errors.Is(err, ErrPaymentState):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_payment_state", "El pago no admite esta operación.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "payment_mutation_failed", "No fue posible guardar el pago.")
	default:
		webutil.WriteJSON(w, status, item)
	}
}

func (h *Handler) ListDeposits(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validDepositStatus(status) {
		webutil.WriteValidationError(w, r, map[string]string{"status": "Estado de depósito no soportado."})
		return
	}
	items, err := h.repository.ListDeposits(r.Context(), webutil.TenantID(r.Context()), status, queryLimit(r, 100))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "deposit_list_failed", "No fue posible cargar los depósitos.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetDeposit(w http.ResponseWriter, r *http.Request) {
	depositID, ok := pathUUID(w, r, "depositID", "invalid_deposit_id", "El ID de depósito es inválido.")
	if !ok {
		return
	}
	item, err := h.repository.GetDeposit(r.Context(), webutil.TenantID(r.Context()), depositID)
	if errors.Is(err, ErrDepositNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "deposit_not_found", "Depósito no encontrado.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "deposit_load_failed", "No fue posible cargar el depósito.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) CreateDeposit(w http.ResponseWriter, r *http.Request) {
	var input CreateDepositInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.CreateDeposit(r.Context(), webutil.TenantID(r.Context()), input)
	h.writeDepositMutation(w, r, item, fields, err, http.StatusCreated)
}

func (h *Handler) ReceiveDeposit(w http.ResponseWriter, r *http.Request) {
	depositID, ok := pathUUID(w, r, "depositID", "invalid_deposit_id", "El ID de depósito es inválido.")
	if !ok {
		return
	}
	var input ReceiveDepositInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.ReceiveDeposit(r.Context(), webutil.TenantID(r.Context()), depositID, input)
	h.writeDepositMutation(w, r, item, fields, err, http.StatusOK)
}

func (h *Handler) SettleDeposit(w http.ResponseWriter, r *http.Request) {
	depositID, ok := pathUUID(w, r, "depositID", "invalid_deposit_id", "El ID de depósito es inválido.")
	if !ok {
		return
	}
	var input SettleDepositInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.SettleDeposit(r.Context(), webutil.TenantID(r.Context()), depositID, input)
	h.writeDepositMutation(w, r, item, fields, err, http.StatusOK)
}

func (h *Handler) writeDepositMutation(w http.ResponseWriter, r *http.Request, item SecurityDeposit, fields map[string]string, err error, status int) {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	switch {
	case errors.Is(err, ErrDepositNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "deposit_not_found", "Depósito no encontrado.")
	case errors.Is(err, ErrSourceNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "deposit_reservation_not_found", "La reserva no existe o fue cancelada.")
	case errors.Is(err, ErrCurrencyMismatch):
		webutil.WriteError(w, r, http.StatusConflict, "deposit_currency_mismatch", "La moneda del depósito debe coincidir con la moneda de la empresa.")
	case errors.Is(err, ErrDepositState):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_deposit_state", "El depósito no admite esta operación o los montos superan su saldo.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "deposit_mutation_failed", "No fue posible guardar el depósito.")
	default:
		webutil.WriteJSON(w, status, item)
	}
}

func pathUUID(w http.ResponseWriter, r *http.Request, name, code, message string) (string, bool) {
	value := r.PathValue(name)
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, code, message)
		return "", false
	}
	return value, true
}

func queryLimit(r *http.Request, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 250 {
		return 250
	}
	return value
}

func validInvoiceStatus(value string) bool {
	switch value {
	case "DRAFT", "ISSUED", "PARTIALLY_PAID", "PAID", "VOID", "OVERDUE":
		return true
	default:
		return false
	}
}

func validDepositStatus(value string) bool {
	switch value {
	case "PENDING", "RECEIVED", "PARTIALLY_SETTLED", "RETURNED", "RETAINED", "SETTLED":
		return true
	default:
		return false
	}
}
