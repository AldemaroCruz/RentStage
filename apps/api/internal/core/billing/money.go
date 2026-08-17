package billing

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	moneyScale    int64 = 100
	maxMoneyCents int64 = 99_999_999_999_999 // NUMERIC(14,2)
)

func moneyCents(value float64) int64 {
	return int64(math.Round(value * float64(moneyScale)))
}

func moneyValue(cents int64) float64 {
	return float64(cents) / float64(moneyScale)
}

func roundMoney(value float64) float64 {
	return moneyValue(moneyCents(value))
}

func roundQuantity(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func calculateLine(
	input InvoiceItemInput,
	rule TaxRule,
	pricesIncludeTax bool,
	sortOrder int,
) (normalizedInvoiceItem, map[string]string) {
	fields := map[string]string{}
	resourceID := cleanOptionalString(input.ResourceID)
	description := strings.TrimSpace(input.Description)
	quantity := roundQuantity(input.Quantity)
	unitPrice := math.Round(input.UnitPrice*10000) / 10000
	discount := roundMoney(input.DiscountAmount)

	if description == "" {
		fields["description"] = "Description is required."
	} else if len(description) > 500 {
		fields["description"] = "Description must be 500 characters or fewer."
	}
	if math.IsNaN(quantity) || math.IsInf(quantity, 0) || quantity <= 0 || quantity > 100000 {
		fields["quantity"] = "Quantity must be greater than zero and no more than 100,000."
	}
	if math.IsNaN(unitPrice) || math.IsInf(unitPrice, 0) || unitPrice < 0 || unitPrice > 9_999_999_999.9999 {
		fields["unit_price"] = "Unit price must be zero or greater and within the supported range."
	}
	if math.IsNaN(discount) || math.IsInf(discount, 0) {
		fields["discount_amount"] = "Discount must be a finite monetary amount."
	}
	grossCents := moneyCents(quantity * unitPrice)
	discountCents := moneyCents(discount)
	if grossCents > maxMoneyCents {
		fields["unit_price"] = "The line total exceeds the supported monetary range."
	}
	if discountCents < 0 || discountCents > grossCents {
		fields["discount_amount"] = "Discount must be between zero and the line gross amount."
	}
	if rule.ID == "" {
		fields["tax_rule_id"] = "Select a valid tax rule."
	}
	if len(fields) > 0 {
		return normalizedInvoiceItem{}, fields
	}

	baseCents := grossCents - discountCents
	netCents := baseCents
	taxCents := int64(0)
	rateBasisPoints := int64(math.Round(rule.Rate * 100))
	if rule.Category == "TAXABLE" && rateBasisPoints > 0 {
		if pricesIncludeTax {
			netCents = int64(math.Round(float64(baseCents*10000) / float64(10000+rateBasisPoints)))
			taxCents = baseCents - netCents
		} else {
			taxCents = int64(math.Round(float64(netCents*rateBasisPoints) / 10000))
		}
	}
	lineTotalCents := netCents + taxCents

	return normalizedInvoiceItem{
		ResourceID:     resourceID,
		TaxRuleID:      stringPointer(rule.ID),
		Description:    description,
		Quantity:       quantity,
		UnitPrice:      unitPrice,
		DiscountAmount: moneyValue(discountCents),
		GrossAmount:    moneyValue(grossCents),
		NetAmount:      moneyValue(netCents),
		TaxCode:        rule.Code,
		TaxCategory:    rule.Category,
		TaxRate:        rule.Rate,
		TaxAmount:      moneyValue(taxCents),
		LineTotal:      moneyValue(lineTotalCents),
		SortOrder:      sortOrder,
	}, nil
}

func aggregateInvoice(items []normalizedInvoiceItem) (taxable, exempt, nonTaxable, tax, total float64) {
	var taxableCents, exemptCents, nonTaxableCents, taxCents, totalCents int64
	for _, item := range items {
		net := moneyCents(item.NetAmount)
		switch item.TaxCategory {
		case "TAXABLE":
			taxableCents += net
		case "EXEMPT":
			exemptCents += net
		case "NON_TAXABLE":
			nonTaxableCents += net
		}
		taxCents += moneyCents(item.TaxAmount)
		totalCents += moneyCents(item.LineTotal)
	}
	return moneyValue(taxableCents), moneyValue(exemptCents), moneyValue(nonTaxableCents), moneyValue(taxCents), moneyValue(totalCents)
}

func allocateHeaderDiscount(items []sourceItem, amount float64) []sourceItem {
	discountCents := moneyCents(amount)
	if discountCents <= 0 || len(items) == 0 {
		return items
	}
	bases := make([]int64, len(items))
	var totalBase int64
	for index, item := range items {
		gross := moneyCents(item.Quantity * item.UnitPrice)
		base := gross - moneyCents(item.DiscountAmount)
		if base < 0 {
			base = 0
		}
		bases[index] = base
		totalBase += base
	}
	if totalBase <= 0 {
		return items
	}
	if discountCents > totalBase {
		discountCents = totalBase
	}

	type remainder struct {
		index int
		value float64
	}
	allocated := make([]int64, len(items))
	remainders := make([]remainder, 0, len(items))
	var allocatedTotal int64
	for index, base := range bases {
		exact := float64(discountCents) * float64(base) / float64(totalBase)
		floorValue := int64(math.Floor(exact))
		allocated[index] = floorValue
		allocatedTotal += floorValue
		remainders = append(remainders, remainder{index: index, value: exact - float64(floorValue)})
	}
	sort.SliceStable(remainders, func(i, j int) bool { return remainders[i].value > remainders[j].value })
	for remaining, cursor := discountCents-allocatedTotal, 0; remaining > 0; remaining-- {
		allocated[remainders[cursor%len(remainders)].index]++
		cursor++
	}

	result := append([]sourceItem(nil), items...)
	for index := range result {
		result[index].DiscountAmount = moneyValue(moneyCents(result[index].DiscountAmount) + allocated[index])
	}
	return result
}

func fiscalProfileMissing(settings Settings) []string {
	checks := []struct {
		name  string
		value string
	}{
		{"legal_name", settings.LegalName},
		{"tax_id", settings.TaxID},
		{"tax_registration_number", settings.TaxRegistrationNumber},
		{"economic_activity", settings.EconomicActivity},
		{"economic_activity_code", settings.EconomicActivityCode},
		{"fiscal_address", settings.FiscalAddress},
		{"department_code", settings.DepartmentCode},
		{"municipality_code", settings.MunicipalityCode},
		{"email", settings.Email},
		{"phone", settings.Phone},
	}
	missing := make([]string, 0)
	for _, check := range checks {
		if strings.TrimSpace(check.value) == "" {
			missing = append(missing, check.name)
		}
	}
	return missing
}

func invoiceDisplayNumber(prefix string, number *int64) string {
	if number == nil {
		return "BORRADOR"
	}
	return fmt.Sprintf("%s-%06d", prefix, *number)
}

func paymentDisplayNumber(number int64) string {
	return fmt.Sprintf("PMT-%06d", number)
}

func depositDisplayNumber(number int64) string {
	return fmt.Sprintf("DEP-%06d", number)
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copy := value
	return &copy
}
