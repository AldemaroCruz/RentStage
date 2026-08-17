package dte

import (
	"strings"
	"testing"
)

func completeInvoiceSnapshot() InvoiceSnapshot {
	return InvoiceSnapshot{
		ID:                         "00000000-0000-0000-0000-000000000111",
		InvoiceNumber:              42,
		InvoicePrefix:              "INV",
		Status:                     "ISSUED",
		FiscalStatus:               "READY_FOR_DTE",
		IssueDate:                  "2026-08-15",
		DueDate:                    "2026-08-15",
		Currency:                   "USD",
		CustomerID:                 "00000000-0000-0000-0000-000000000222",
		CustomerName:               "Cliente Demo",
		CustomerTaxID:              "06141234567890",
		CustomerRegistrationNumber: "1234567",
		CustomerDocumentType:       "36",
		CustomerTradeName:          "Cliente Demo",
		CustomerEconomicActivity:   "Servicios para eventos",
		CustomerEconomicCode:       "90001",
		CustomerEmail:              "cliente@example.com",
		CustomerPhone:              "+50370000000",
		CustomerAddress:            "San Salvador",
		CustomerDepartmentCode:     "06",
		CustomerMunicipalityCode:   "14",
		SellerLegalName:            "AudioPro Demo, S.A. de C.V.",
		SellerTradeName:            "AudioPro Demo",
		SellerTaxID:                "06140000000000",
		SellerRegistrationNumber:   "1000000",
		SellerEconomicActivity:     "Alquiler de equipo de audio",
		SellerEconomicCode:         "77300",
		SellerEmail:                "facturacion@audiopro.example",
		SellerPhone:                "+50322000000",
		SellerAddress:              "San Salvador",
		SellerDepartmentCode:       "06",
		SellerMunicipalityCode:     "14",
		TaxableAmount:              100,
		TaxAmount:                  13,
		TotalAmount:                113,
		Items: []InvoiceItemSnapshot{{
			ID:          "00000000-0000-0000-0000-000000000333",
			Description: "Servicio de audio",
			Quantity:    1,
			UnitPrice:   100,
			NetAmount:   100,
			TaxCategory: "TAXABLE",
			TaxRate:     13,
			TaxAmount:   13,
			LineTotal:   113,
			DTEItemType: 2,
			DTEUnitCode: 59,
		}},
	}
}

func completeSettings() Settings {
	return Settings{
		Enabled:             true,
		ProviderMode:        "MOCK",
		Environment:         "TEST",
		DefaultDocumentType: "01",
		SchemaVersion:       1,
		EstablishmentType:   "01",
		EstablishmentCode:   "M001",
		PointOfSaleCode:     "P001",
	}
}

func TestControlNumber(t *testing.T) {
	got := controlNumber("01", "m001", "p001", 42)
	want := "DTE-01-M001P001-000000000000042"
	if got != want {
		t.Fatalf("control number = %q, want %q", got, want)
	}
	if sequenceFromControl(got) != 42 {
		t.Fatalf("sequenceFromControl(%q) did not return 42", got)
	}
}

func TestNormalizeDocumentTypeChoosesCCFForRegisteredCustomer(t *testing.T) {
	invoice := completeInvoiceSnapshot()
	if got := normalizeDocumentType("", completeSettings(), invoice); got != "03" {
		t.Fatalf("normalizeDocumentType = %q, want 03", got)
	}
	if got := normalizeDocumentType("01", completeSettings(), invoice); got != "01" {
		t.Fatalf("explicit document type was not preserved: %q", got)
	}
}

func TestValidateSnapshotForFacturaAndCCF(t *testing.T) {
	invoice := completeInvoiceSnapshot()
	if fields := validateSnapshot(completeSettings(), invoice, "03"); len(fields) != 0 {
		t.Fatalf("complete CCF snapshot returned validation fields: %#v", fields)
	}

	invoice.CustomerRegistrationNumber = ""
	invoice.CustomerEconomicCode = ""
	fields := validateSnapshot(completeSettings(), invoice, "03")
	if fields["customer_registration_number"] == "" || fields["customer_economic_activity"] == "" {
		t.Fatalf("CCF receiver requirements were not enforced: %#v", fields)
	}
}

func TestValidateSnapshotRequiresReceiverForLargeFactura(t *testing.T) {
	invoice := completeInvoiceSnapshot()
	invoice.TotalAmount = 1095.01
	invoice.CustomerTaxID = ""
	invoice.CustomerAddress = ""
	invoice.CustomerDepartmentCode = ""
	invoice.CustomerMunicipalityCode = ""
	fields := validateSnapshot(completeSettings(), invoice, "01")
	if fields["customer_tax_id"] == "" || fields["customer_address"] == "" {
		t.Fatalf("large factura receiver requirements were not enforced: %#v", fields)
	}
}

func TestBuildPayloadFreezesCoreFiscalValues(t *testing.T) {
	settings := completeSettings()
	invoice := completeInvoiceSnapshot()
	control := controlNumber("01", settings.EstablishmentCode, settings.PointOfSaleCode, 42)
	payload := buildPayload(settings, invoice, "01", "E5AA7B13-0675-4AFA-917A-8A9ACEDDA281", control)

	identification, ok := payload["identificacion"].(map[string]any)
	if !ok {
		t.Fatalf("payload identification is missing")
	}
	if identification["numeroControl"] != control || identification["tipoDte"] != "01" {
		t.Fatalf("unexpected identification: %#v", identification)
	}

	body, ok := payload["cuerpoDocumento"].([]map[string]any)
	if !ok || len(body) != 1 {
		t.Fatalf("unexpected document body: %#v", payload["cuerpoDocumento"])
	}
	if body[0]["ventaGravada"] != float64(100) || body[0]["ivaItem"] != float64(13) {
		t.Fatalf("line fiscal values were not preserved: %#v", body[0])
	}

	summary, ok := payload["resumen"].(map[string]any)
	if !ok || summary["totalPagar"] != float64(113) || summary["totalIva"] != float64(13) {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if words, _ := summary["totalLetras"].(string); !strings.Contains(words, "CIENTO TRECE") {
		t.Fatalf("unexpected amount in words: %q", words)
	}
}

func TestNormalizeSettingsRejectsUnsafeMockProduction(t *testing.T) {
	input := SettingsInput{
		Enabled:             true,
		ProviderMode:        "MOCK",
		Environment:         "PRODUCTION",
		DefaultDocumentType: "01",
		SchemaVersion:       1,
		EstablishmentType:   "01",
		EstablishmentCode:   "M001",
		PointOfSaleCode:     "P001",
		MaxAttempts:         5,
		RetryBaseSeconds:    60,
		AutoSubmitOnIssue:   true,
	}
	normalized, fields := normalizeSettings(input)
	if fields["environment"] == "" || fields["auto_submit_on_issue"] == "" {
		t.Fatalf("unsafe settings were not rejected: %#v", fields)
	}
	if normalized.AutoSubmitOnIssue {
		t.Fatalf("unsupported auto-submit remained enabled")
	}
}
