package billing

import "testing"

func TestNormalizeInvoiceBaseCalculatesTotals(t *testing.T) {
	ruleID := "11111111-1111-1111-1111-111111111111"
	invoice, fields := normalizeInvoiceBase(
		"22222222-2222-2222-2222-222222222222",
		"2026-08-15",
		"2026-08-20",
		"USD",
		false,
		"",
		"",
		[]InvoiceItemInput{{
			TaxRuleID:   ruleID,
			Description: "Servicio de audio",
			Quantity:    1,
			UnitPrice:   100,
		}},
		[]TaxRule{{
			ID: ruleID, Code: "IVA", Name: "IVA estándar", Category: "TAXABLE", Rate: 13, Active: true, IsDefault: true,
		}},
		0,
	)
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if invoice.TaxableAmount != 100 || invoice.TaxAmount != 13 || invoice.TotalAmount != 113 {
		t.Fatalf("unexpected invoice totals: %#v", invoice)
	}
}

func TestNormalizePaymentRequiresExactAllocations(t *testing.T) {
	_, fields := normalizePayment(CreatePaymentInput{
		CustomerID: "22222222-2222-2222-2222-222222222222",
		Amount:     75,
		Currency:   "USD",
		Method:     "BANK_TRANSFER",
		Allocations: []PaymentAllocationInput{{
			InvoiceID: "33333333-3333-3333-3333-333333333333",
			Amount:    50,
		}},
	})
	if fields["allocations"] == "" {
		t.Fatalf("expected exact-allocation validation, got %#v", fields)
	}
}

func TestNormalizeSettingsDefaultsInvoicePrefix(t *testing.T) {
	settings, fields := normalizeSettings(SettingsInput{
		DefaultTaxRate:          13,
		DefaultPaymentTermsDays: 15,
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if settings.InvoicePrefix != "INV" {
		t.Fatalf("expected INV prefix, got %q", settings.InvoicePrefix)
	}
}
