package billing

import (
	"context"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

var invoicePrefixPattern = regexp.MustCompile(`^[A-Z0-9_-]{1,12}$`)

const dateLayout = "2006-01-02"

type Service struct {
	repository *Repository
	audit      *audit.Repository
}

func NewService(repository *Repository, auditRepository *audit.Repository) *Service {
	return &Service{repository: repository, audit: auditRepository}
}

func (s *Service) UpdateSettings(ctx context.Context, tenantID string, input SettingsInput) (Settings, map[string]string, error) {
	normalized, fields := normalizeSettings(input)
	if len(fields) > 0 {
		return Settings{}, fields, nil
	}
	item, err := s.repository.UpdateSettings(ctx, tenantID, normalized)
	if err != nil {
		return Settings{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "BILLING_SETTINGS_UPDATED", "billing_settings", nil, map[string]any{
		"enabled":                 item.Enabled,
		"prices_include_tax":      item.PricesIncludeTax,
		"default_tax_rate":        item.DefaultTaxRate,
		"invoice_prefix":          item.InvoicePrefix,
		"fiscal_profile_complete": item.FiscalProfileComplete,
		"fiscal_profile_missing":  item.FiscalProfileMissingFields,
	})
	return item, nil, nil
}

func normalizeSettings(input SettingsInput) (SettingsInput, map[string]string) {
	result := SettingsInput{
		Enabled:                 input.Enabled,
		LegalName:               strings.TrimSpace(input.LegalName),
		TradeName:               strings.TrimSpace(input.TradeName),
		TaxID:                   strings.TrimSpace(input.TaxID),
		TaxRegistrationNumber:   strings.TrimSpace(input.TaxRegistrationNumber),
		EconomicActivity:        strings.TrimSpace(input.EconomicActivity),
		EconomicActivityCode:    strings.TrimSpace(input.EconomicActivityCode),
		FiscalAddress:           strings.TrimSpace(input.FiscalAddress),
		Department:              strings.TrimSpace(input.Department),
		Municipality:            strings.TrimSpace(input.Municipality),
		District:                strings.TrimSpace(input.District),
		DepartmentCode:          strings.TrimSpace(input.DepartmentCode),
		MunicipalityCode:        strings.TrimSpace(input.MunicipalityCode),
		DistrictCode:            strings.TrimSpace(input.DistrictCode),
		Email:                   strings.ToLower(strings.TrimSpace(input.Email)),
		Phone:                   strings.TrimSpace(input.Phone),
		PricesIncludeTax:        input.PricesIncludeTax,
		DefaultTaxRate:          roundMoney(input.DefaultTaxRate),
		DefaultPaymentTermsDays: input.DefaultPaymentTermsDays,
		InvoicePrefix:           strings.ToUpper(strings.TrimSpace(input.InvoicePrefix)),
	}
	if result.InvoicePrefix == "" {
		result.InvoicePrefix = "INV"
	}
	fields := map[string]string{}
	max := func(name, value string, length int) {
		if len(value) > length {
			fields[name] = "El valor supera el máximo permitido."
		}
	}
	max("legal_name", result.LegalName, 240)
	max("trade_name", result.TradeName, 240)
	max("tax_id", result.TaxID, 40)
	max("tax_registration_number", result.TaxRegistrationNumber, 40)
	max("economic_activity", result.EconomicActivity, 240)
	max("economic_activity_code", result.EconomicActivityCode, 40)
	max("fiscal_address", result.FiscalAddress, 4000)
	max("department", result.Department, 120)
	max("municipality", result.Municipality, 120)
	max("district", result.District, 120)
	max("department_code", result.DepartmentCode, 4)
	max("municipality_code", result.MunicipalityCode, 4)
	max("district_code", result.DistrictCode, 8)
	max("phone", result.Phone, 80)
	if result.Email != "" {
		parsed, err := mail.ParseAddress(result.Email)
		if err != nil || !strings.EqualFold(parsed.Address, result.Email) {
			fields["email"] = "Ingresa un correo válido."
		}
	}
	if result.DefaultTaxRate < 0 || result.DefaultTaxRate > 100 {
		fields["default_tax_rate"] = "La tasa debe estar entre 0 y 100."
	}
	if result.DefaultPaymentTermsDays < 0 || result.DefaultPaymentTermsDays > 365 {
		fields["default_payment_terms_days"] = "Los días de crédito deben estar entre 0 y 365."
	}
	if !invoicePrefixPattern.MatchString(result.InvoicePrefix) {
		fields["invoice_prefix"] = "Usa entre 1 y 12 letras, números, guion o guion bajo."
	}
	return result, fields
}

func (s *Service) CreateInvoice(ctx context.Context, tenantID string, input CreateInvoiceInput) (InvoiceDetail, map[string]string, error) {
	settings, err := s.repository.GetSettings(ctx, tenantID)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	if !settings.Enabled {
		return InvoiceDetail{}, nil, ErrBillingDisabled
	}
	tenantCurrency, err := s.repository.TenantCurrency(ctx, tenantID)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	rules, err := s.repository.ListTaxRules(ctx, tenantID)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}

	sourceType := strings.ToUpper(strings.TrimSpace(input.SourceType))
	if sourceType == "" {
		sourceType = "MANUAL"
	}
	fields := map[string]string{}
	var source sourceDraft
	var items []InvoiceItemInput
	customerID := strings.TrimSpace(input.CustomerID)
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	notes := strings.TrimSpace(input.Notes)
	terms := strings.TrimSpace(input.Terms)

	switch sourceType {
	case "MANUAL":
		items = input.Items
	case "QUOTE", "RESERVATION":
		if input.SourceID == nil || !idutil.IsUUID(strings.TrimSpace(*input.SourceID)) {
			fields["source_id"] = "Selecciona una cotización o reserva válida."
			return InvoiceDetail{}, fields, nil
		}
		source, err = s.repository.LoadSource(ctx, tenantID, sourceType, strings.TrimSpace(*input.SourceID))
		if err != nil {
			return InvoiceDetail{}, nil, err
		}
		customerID = source.CustomerID
		if currency == "" {
			currency = source.Currency
		}
		if notes == "" {
			notes = source.Notes
		}
		defaultRule, ok := defaultTaxRule(rules)
		if !ok {
			return InvoiceDetail{}, nil, ErrTaxRuleNotFound
		}
		sourceItems := allocateHeaderDiscount(source.Items, source.HeaderDiscount)
		items = make([]InvoiceItemInput, 0, len(sourceItems)+1)
		for _, line := range sourceItems {
			items = append(items, InvoiceItemInput{
				ResourceID:     line.ResourceID,
				TaxRuleID:      defaultRule.ID,
				Description:    line.Description,
				Quantity:       line.Quantity,
				UnitPrice:      line.UnitPrice,
				DiscountAmount: line.DiscountAmount,
			})
		}
		if moneyCents(source.ExtraCharges) > 0 {
			items = append(items, InvoiceItemInput{
				TaxRuleID:   defaultRule.ID,
				Description: "Cargos adicionales",
				Quantity:    1,
				UnitPrice:   roundMoney(source.ExtraCharges),
			})
		}
	default:
		fields["source_type"] = "El origen debe ser MANUAL, QUOTE o RESERVATION."
		return InvoiceDetail{}, fields, nil
	}
	if currency == "" {
		currency = tenantCurrency
	}
	if currency != tenantCurrency {
		fields["currency"] = "La moneda debe coincidir con la moneda de la empresa (" + tenantCurrency + ")."
	}

	normalized, normalizationFields := normalizeInvoiceBase(
		customerID,
		input.IssueDate,
		input.DueDate,
		currency,
		settings.PricesIncludeTax,
		notes,
		terms,
		items,
		rules,
		settings.DefaultPaymentTermsDays,
	)
	mergeFields(fields, normalizationFields)
	if len(fields) > 0 {
		return InvoiceDetail{}, fields, nil
	}
	normalized.SourceType = sourceType
	if sourceType == "QUOTE" {
		normalized.QuoteID = source.QuoteID
		normalized.ReservationID = source.ReservationID
	}
	if sourceType == "RESERVATION" {
		normalized.ReservationID = source.ReservationID
		normalized.QuoteID = source.QuoteID
	}

	item, err := s.repository.CreateInvoice(ctx, tenantID, normalized, settings)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "INVOICE_CREATED", "invoice", &item.ID, map[string]any{
		"source_type": item.SourceType,
		"customer":    item.CustomerName,
		"total":       item.TotalAmount,
	})
	return item, nil, nil
}

func (s *Service) UpdateInvoice(ctx context.Context, tenantID, invoiceID string, input UpdateInvoiceInput) (InvoiceDetail, map[string]string, error) {
	settings, err := s.repository.GetSettings(ctx, tenantID)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	tenantCurrency, err := s.repository.TenantCurrency(ctx, tenantID)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	rules, err := s.repository.ListTaxRules(ctx, tenantID)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	current, err := s.repository.GetInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	normalized, fields := normalizeInvoiceBase(
		input.CustomerID,
		input.IssueDate,
		input.DueDate,
		input.Currency,
		input.PricesIncludeTax,
		input.Notes,
		input.Terms,
		input.Items,
		rules,
		settings.DefaultPaymentTermsDays,
	)
	if normalized.Currency != tenantCurrency {
		fields["currency"] = "La moneda debe coincidir con la moneda de la empresa (" + tenantCurrency + ")."
	}
	if len(fields) > 0 {
		return InvoiceDetail{}, fields, nil
	}
	normalized.SourceType = current.SourceType
	normalized.QuoteID = current.QuoteID
	normalized.ReservationID = current.ReservationID
	item, err := s.repository.UpdateInvoice(ctx, tenantID, invoiceID, normalized, settings)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "INVOICE_UPDATED", "invoice", &item.ID, map[string]any{
		"total": item.TotalAmount,
	})
	return item, nil, nil
}

func normalizeInvoiceBase(
	customerID, issueDateRaw, dueDateRaw, currency string,
	pricesIncludeTax bool,
	notes, terms string,
	items []InvoiceItemInput,
	rules []TaxRule,
	defaultTerms int,
) (normalizedInvoice, map[string]string) {
	result := normalizedInvoice{
		CustomerID:       strings.TrimSpace(customerID),
		Currency:         strings.ToUpper(strings.TrimSpace(currency)),
		PricesIncludeTax: pricesIncludeTax,
		Notes:            strings.TrimSpace(notes),
		Terms:            strings.TrimSpace(terms),
		Items:            make([]normalizedInvoiceItem, 0, len(items)),
	}
	fields := map[string]string{}
	if !idutil.IsUUID(result.CustomerID) {
		fields["customer_id"] = "Selecciona un cliente válido."
	}
	if result.Currency == "" {
		result.Currency = "USD"
	}
	if len(result.Currency) != 3 {
		fields["currency"] = "La moneda debe usar un código ISO de 3 letras."
	}

	issueDate, err := parseDate(issueDateRaw)
	if err != nil {
		fields["issue_date"] = "Usa una fecha válida en formato YYYY-MM-DD."
	} else {
		result.IssueDate = issueDate
	}
	if strings.TrimSpace(dueDateRaw) == "" && err == nil {
		result.DueDate = issueDate.AddDate(0, 0, defaultTerms)
	} else {
		dueDate, dueErr := parseDate(dueDateRaw)
		if dueErr != nil {
			fields["due_date"] = "Usa una fecha válida en formato YYYY-MM-DD."
		} else {
			result.DueDate = dueDate
		}
	}
	if !result.IssueDate.IsZero() && !result.DueDate.IsZero() && result.DueDate.Before(result.IssueDate) {
		fields["due_date"] = "La fecha de vencimiento no puede ser anterior a la emisión."
	}
	if len(result.Notes) > 8000 {
		fields["notes"] = "Las notas no pueden superar 8,000 caracteres."
	}
	if len(result.Terms) > 12000 {
		fields["terms"] = "Los términos no pueden superar 12,000 caracteres."
	}
	if len(items) == 0 {
		fields["items"] = "Agrega al menos una línea."
	}
	if len(items) > 200 {
		fields["items"] = "Una factura puede contener como máximo 200 líneas."
	}
	ruleMap := make(map[string]TaxRule, len(rules))
	for _, rule := range rules {
		ruleMap[rule.ID] = rule
	}
	for index, input := range items {
		prefix := "items[" + intString(index) + "]"
		rule, ok := ruleMap[strings.TrimSpace(input.TaxRuleID)]
		if !ok {
			fields[prefix+".tax_rule_id"] = "Selecciona una regla tributaria válida."
			continue
		}
		item, itemFields := calculateLine(input, rule, pricesIncludeTax, index)
		for field, message := range itemFields {
			fields[prefix+"."+field] = message
		}
		if len(itemFields) == 0 {
			result.Items = append(result.Items, item)
		}
	}
	if len(fields) == 0 {
		result.TaxableAmount, result.ExemptAmount, result.NonTaxableAmount,
			result.TaxAmount, result.TotalAmount = aggregateInvoice(result.Items)
		if moneyCents(result.TotalAmount) > maxMoneyCents {
			fields["items"] = "El total de la factura supera el rango monetario permitido."
		}
	}
	return result, fields
}

func (s *Service) IssueInvoice(ctx context.Context, tenantID, invoiceID string) (InvoiceDetail, error) {
	item, err := s.repository.IssueInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "INVOICE_ISSUED", "invoice", &item.ID, map[string]any{
		"invoice_number": item.DisplayNumber,
		"total":          item.TotalAmount,
		"fiscal_status":  item.FiscalStatus,
	})
	return item, nil
}

func (s *Service) VoidInvoice(ctx context.Context, tenantID, invoiceID string, input VoidInput) (InvoiceDetail, map[string]string, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return InvoiceDetail{}, map[string]string{"reason": "Indica el motivo de anulación."}, nil
	}
	if len(reason) > 2000 {
		return InvoiceDetail{}, map[string]string{"reason": "El motivo no puede superar 2,000 caracteres."}, nil
	}
	item, err := s.repository.VoidInvoice(ctx, tenantID, invoiceID, reason)
	if err != nil {
		return InvoiceDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "INVOICE_VOIDED", "invoice", &item.ID, map[string]any{
		"invoice_number": item.DisplayNumber,
		"reason":         reason,
	})
	return item, nil, nil
}

func (s *Service) CreatePayment(ctx context.Context, tenantID string, input CreatePaymentInput) (PaymentDetail, map[string]string, error) {
	tenantCurrency, err := s.repository.TenantCurrency(ctx, tenantID)
	if err != nil {
		return PaymentDetail{}, nil, err
	}
	if strings.TrimSpace(input.Currency) == "" {
		input.Currency = tenantCurrency
	}
	normalized, fields := normalizePayment(input)
	if normalized.Currency != tenantCurrency {
		fields["currency"] = "La moneda debe coincidir con la moneda de la empresa (" + tenantCurrency + ")."
	}
	if len(fields) > 0 {
		return PaymentDetail{}, fields, nil
	}
	item, err := s.repository.CreatePayment(ctx, tenantID, normalized)
	if err != nil {
		return PaymentDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "PAYMENT_RECORDED", "payment", &item.ID, map[string]any{
		"payment_number": item.DisplayNumber,
		"customer":       item.CustomerName,
		"amount":         item.Amount,
		"allocations":    item.AllocationCount,
	})
	return item, nil, nil
}

func normalizePayment(input CreatePaymentInput) (normalizedPayment, map[string]string) {
	result := normalizedPayment{
		CustomerID:  strings.TrimSpace(input.CustomerID),
		Amount:      roundMoney(input.Amount),
		Currency:    strings.ToUpper(strings.TrimSpace(input.Currency)),
		Method:      strings.ToUpper(strings.TrimSpace(input.Method)),
		Reference:   strings.TrimSpace(input.Reference),
		Notes:       strings.TrimSpace(input.Notes),
		Allocations: append([]PaymentAllocationInput(nil), input.Allocations...),
	}
	fields := map[string]string{}
	if !idutil.IsUUID(result.CustomerID) {
		fields["customer_id"] = "Selecciona un cliente válido."
	}
	if result.Amount <= 0 || moneyCents(result.Amount) > maxMoneyCents {
		fields["amount"] = "El monto debe ser mayor que cero y estar dentro del rango permitido."
	}
	if result.Currency == "" {
		result.Currency = "USD"
	}
	if len(result.Currency) != 3 {
		fields["currency"] = "La moneda debe usar un código ISO de 3 letras."
	}
	if !validPaymentMethod(result.Method) {
		fields["method"] = "Selecciona un método de pago válido."
	}
	if len(result.Reference) > 240 {
		fields["reference"] = "La referencia no puede superar 240 caracteres."
	}
	if len(result.Notes) > 4000 {
		fields["notes"] = "Las notas no pueden superar 4,000 caracteres."
	}
	if strings.TrimSpace(input.ReceivedAt) == "" {
		result.ReceivedAt = time.Now().UTC()
	} else {
		value, err := time.Parse(time.RFC3339, strings.TrimSpace(input.ReceivedAt))
		if err != nil {
			fields["received_at"] = "Usa una fecha y hora RFC3339 válida."
		} else {
			result.ReceivedAt = value
		}
	}
	if len(result.Allocations) == 0 {
		fields["allocations"] = "Asigna el pago al menos a una factura."
	}
	if len(result.Allocations) > 100 {
		fields["allocations"] = "Un pago puede distribuirse entre un máximo de 100 facturas."
	}
	seen := map[string]struct{}{}
	var totalCents int64
	for index := range result.Allocations {
		allocation := &result.Allocations[index]
		allocation.InvoiceID = strings.TrimSpace(allocation.InvoiceID)
		allocation.Amount = roundMoney(allocation.Amount)
		prefix := "allocations[" + intString(index) + "]"
		if !idutil.IsUUID(allocation.InvoiceID) {
			fields[prefix+".invoice_id"] = "La factura es inválida."
		}
		if _, exists := seen[allocation.InvoiceID]; exists {
			fields[prefix+".invoice_id"] = "Cada factura solo puede aparecer una vez."
		}
		seen[allocation.InvoiceID] = struct{}{}
		if allocation.Amount <= 0 {
			fields[prefix+".amount"] = "El monto asignado debe ser mayor que cero."
		}
		totalCents += moneyCents(allocation.Amount)
	}
	if moneyCents(result.Amount) != totalCents {
		fields["allocations"] = "La suma de asignaciones debe coincidir exactamente con el monto del pago."
	}
	return result, fields
}

func (s *Service) VoidPayment(ctx context.Context, tenantID, paymentID string, input VoidInput) (PaymentDetail, map[string]string, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return PaymentDetail{}, map[string]string{"reason": "Indica el motivo de reversión."}, nil
	}
	item, err := s.repository.VoidPayment(ctx, tenantID, paymentID, reason)
	if err != nil {
		return PaymentDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "PAYMENT_VOIDED", "payment", &item.ID, map[string]any{
		"payment_number": item.DisplayNumber,
		"amount":         item.Amount,
		"reason":         reason,
	})
	return item, nil, nil
}

func (s *Service) CreateDeposit(ctx context.Context, tenantID string, input CreateDepositInput) (SecurityDeposit, map[string]string, error) {
	tenantCurrency, err := s.repository.TenantCurrency(ctx, tenantID)
	if err != nil {
		return SecurityDeposit{}, nil, err
	}
	input.ReservationID = strings.TrimSpace(input.ReservationID)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.Currency == "" {
		input.Currency = tenantCurrency
	}
	input.Method = strings.ToUpper(strings.TrimSpace(input.Method))
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)
	fields := map[string]string{}
	if !idutil.IsUUID(input.ReservationID) {
		fields["reservation_id"] = "Selecciona una reserva válida."
	}
	input.Amount = roundMoney(input.Amount)
	if input.Amount <= 0 || moneyCents(input.Amount) > maxMoneyCents {
		fields["amount"] = "El depósito debe ser mayor que cero y estar dentro del rango permitido."
	}
	if len(input.Currency) != 3 {
		fields["currency"] = "La moneda debe usar un código ISO de 3 letras."
	} else if input.Currency != tenantCurrency {
		fields["currency"] = "La moneda debe coincidir con la moneda de la empresa (" + tenantCurrency + ")."
	}
	if input.Method == "" {
		input.Method = "OTHER"
	}
	if !validPaymentMethod(input.Method) {
		fields["method"] = "Selecciona un método válido."
	}
	if len(input.Reference) > 240 {
		fields["reference"] = "La referencia no puede superar 240 caracteres."
	}
	if len(input.Notes) > 4000 {
		fields["notes"] = "Las notas no pueden superar 4,000 caracteres."
	}
	var receivedAt *time.Time
	if input.MarkReceived {
		value := time.Now().UTC()
		if input.ReceivedAt != nil && strings.TrimSpace(*input.ReceivedAt) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*input.ReceivedAt))
			if err != nil {
				fields["received_at"] = "Usa una fecha y hora RFC3339 válida."
			} else {
				value = parsed
			}
		}
		receivedAt = &value
	}
	if len(fields) > 0 {
		return SecurityDeposit{}, fields, nil
	}
	item, err := s.repository.CreateDeposit(ctx, tenantID, input, receivedAt)
	if err != nil {
		return SecurityDeposit{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "SECURITY_DEPOSIT_CREATED", "security_deposit", &item.ID, map[string]any{
		"deposit_number": item.DisplayNumber,
		"reservation":    item.ReservationNumber,
		"amount":         item.Amount,
		"status":         item.Status,
	})
	return item, nil, nil
}

func (s *Service) ReceiveDeposit(ctx context.Context, tenantID, depositID string, input ReceiveDepositInput) (SecurityDeposit, map[string]string, error) {
	input.Method = strings.ToUpper(strings.TrimSpace(input.Method))
	input.Reference = strings.TrimSpace(input.Reference)
	input.Notes = strings.TrimSpace(input.Notes)
	fields := map[string]string{}
	if input.Method == "" {
		input.Method = "OTHER"
	}
	if !validPaymentMethod(input.Method) {
		fields["method"] = "Selecciona un método válido."
	}
	receivedAt := time.Now().UTC()
	if strings.TrimSpace(input.ReceivedAt) != "" {
		value, err := time.Parse(time.RFC3339, strings.TrimSpace(input.ReceivedAt))
		if err != nil {
			fields["received_at"] = "Usa una fecha y hora RFC3339 válida."
		} else {
			receivedAt = value
		}
	}
	if len(fields) > 0 {
		return SecurityDeposit{}, fields, nil
	}
	item, err := s.repository.ReceiveDeposit(ctx, tenantID, depositID, input, receivedAt)
	if err != nil {
		return SecurityDeposit{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "SECURITY_DEPOSIT_RECEIVED", "security_deposit", &item.ID, map[string]any{
		"deposit_number": item.DisplayNumber,
		"amount":         item.Amount,
	})
	return item, nil, nil
}

func (s *Service) SettleDeposit(ctx context.Context, tenantID, depositID string, input SettleDepositInput) (SecurityDeposit, map[string]string, error) {
	input.ReturnedAmount = roundMoney(input.ReturnedAmount)
	input.RetainedAmount = roundMoney(input.RetainedAmount)
	input.Reason = strings.TrimSpace(input.Reason)
	fields := map[string]string{}
	if input.ReturnedAmount < 0 {
		fields["returned_amount"] = "El monto devuelto no puede ser negativo."
	}
	if input.RetainedAmount < 0 {
		fields["retained_amount"] = "El monto retenido no puede ser negativo."
	}
	if input.RetainedAmount > 0 && input.Reason == "" {
		fields["reason"] = "Describe por qué se retiene una parte del depósito."
	}
	settledAt := time.Now().UTC()
	if strings.TrimSpace(input.SettledAt) != "" {
		value, err := time.Parse(time.RFC3339, strings.TrimSpace(input.SettledAt))
		if err != nil {
			fields["settled_at"] = "Usa una fecha y hora RFC3339 válida."
		} else {
			settledAt = value
		}
	}
	if len(fields) > 0 {
		return SecurityDeposit{}, fields, nil
	}
	item, err := s.repository.SettleDeposit(ctx, tenantID, depositID, input, settledAt)
	if err != nil {
		return SecurityDeposit{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "SECURITY_DEPOSIT_SETTLED", "security_deposit", &item.ID, map[string]any{
		"deposit_number": item.DisplayNumber,
		"returned":       item.ReturnedAmount,
		"retained":       item.RetainedAmount,
		"status":         item.Status,
	})
	return item, nil, nil
}

func defaultTaxRule(rules []TaxRule) (TaxRule, bool) {
	for _, rule := range rules {
		if rule.IsDefault && rule.Active {
			return rule, true
		}
	}
	for _, rule := range rules {
		if rule.Active && rule.Category == "TAXABLE" {
			return rule, true
		}
	}
	return TaxRule{}, false
}

func parseDate(value string) (time.Time, error) {
	return time.Parse(dateLayout, strings.TrimSpace(value))
}

func validPaymentMethod(value string) bool {
	switch value {
	case "CASH", "BANK_TRANSFER", "CARD", "CHECK", "OTHER":
		return true
	default:
		return false
	}
}

func mergeFields(target, source map[string]string) {
	for field, message := range source {
		target[field] = message
	}
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}
