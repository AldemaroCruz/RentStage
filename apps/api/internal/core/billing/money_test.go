package billing

import "testing"

func TestCalculateLineTaxExclusive(t *testing.T) {
	item, fields := calculateLine(InvoiceItemInput{
		TaxRuleID:   "11111111-1111-1111-1111-111111111111",
		Description: "Servicio de audio",
		Quantity:    1,
		UnitPrice:   100,
	}, TaxRule{ID: "11111111-1111-1111-1111-111111111111", Code: "IVA", Category: "TAXABLE", Rate: 13}, false, 0)
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if item.NetAmount != 100 || item.TaxAmount != 13 || item.LineTotal != 113 {
		t.Fatalf("unexpected exclusive calculation: %#v", item)
	}
}

func TestCalculateLineTaxIncluded(t *testing.T) {
	item, fields := calculateLine(InvoiceItemInput{
		TaxRuleID:   "11111111-1111-1111-1111-111111111111",
		Description: "Servicio de audio",
		Quantity:    1,
		UnitPrice:   113,
	}, TaxRule{ID: "11111111-1111-1111-1111-111111111111", Code: "IVA", Category: "TAXABLE", Rate: 13}, true, 0)
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if item.NetAmount != 100 || item.TaxAmount != 13 || item.LineTotal != 113 {
		t.Fatalf("unexpected included calculation: %#v", item)
	}
}

func TestCalculateLineExempt(t *testing.T) {
	item, fields := calculateLine(InvoiceItemInput{
		TaxRuleID:   "22222222-2222-2222-2222-222222222222",
		Description: "Operación exenta",
		Quantity:    2,
		UnitPrice:   25,
	}, TaxRule{ID: "22222222-2222-2222-2222-222222222222", Code: "EXEMPT", Category: "EXEMPT", Rate: 0}, false, 0)
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if item.NetAmount != 50 || item.TaxAmount != 0 || item.LineTotal != 50 {
		t.Fatalf("unexpected exempt calculation: %#v", item)
	}
}

func TestAllocateHeaderDiscountPreservesExactAmount(t *testing.T) {
	items := []sourceItem{
		{Description: "A", Quantity: 1, UnitPrice: 10},
		{Description: "B", Quantity: 1, UnitPrice: 10},
		{Description: "C", Quantity: 1, UnitPrice: 10},
	}
	result := allocateHeaderDiscount(items, 1)
	var total int64
	for _, item := range result {
		total += moneyCents(item.DiscountAmount)
	}
	if total != 100 {
		t.Fatalf("discount allocation lost cents: got %d", total)
	}
}

func TestCalculateLineRejectsDatabaseOverflow(t *testing.T) {
	_, fields := calculateLine(InvoiceItemInput{
		TaxRuleID:   "11111111-1111-1111-1111-111111111111",
		Description: "Overflow",
		Quantity:    100000,
		UnitPrice:   9999999999,
	}, TaxRule{ID: "11111111-1111-1111-1111-111111111111", Code: "IVA", Category: "TAXABLE", Rate: 13}, false, 0)
	if fields["unit_price"] == "" {
		t.Fatalf("expected monetary range validation, got %#v", fields)
	}
}
