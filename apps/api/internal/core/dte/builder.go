package dte

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func documentTypeLabel(value string) string {
	switch value {
	case "01":
		return "Factura"
	case "03":
		return "Comprobante de crédito fiscal"
	default:
		return value
	}
}

func invoiceDisplayNumber(prefix string, number *int64) string {
	if number == nil {
		return "BORRADOR"
	}
	return fmt.Sprintf("%s-%06d", prefix, *number)
}

func controlNumber(documentType, establishmentCode, pointOfSaleCode string, sequence int64) string {
	return fmt.Sprintf("DTE-%s-%s%s-%015d",
		documentType,
		strings.ToUpper(establishmentCode),
		strings.ToUpper(pointOfSaleCode),
		sequence,
	)
}

func environmentCode(environment string) string {
	if environment == "PRODUCTION" {
		return "01"
	}
	return "00"
}

func normalizeDocumentType(requested string, settings Settings, invoice InvoiceSnapshot) string {
	value := strings.TrimSpace(requested)
	if value == "01" || value == "03" {
		return value
	}
	// A registered IVA customer with economic activity is eligible for CCF.
	if invoice.CustomerTaxID != "" && invoice.CustomerRegistrationNumber != "" && invoice.CustomerEconomicCode != "" {
		return "03"
	}
	return settings.DefaultDocumentType
}

func validateSnapshot(settings Settings, invoice InvoiceSnapshot, documentType string) map[string]string {
	fields := map[string]string{}
	if invoice.Status == "DRAFT" || invoice.Status == "VOID" || invoice.InvoiceNumber <= 0 {
		fields["invoice"] = "La factura debe estar emitida y vigente antes de preparar un DTE."
	}
	if invoice.FiscalStatus != "READY_FOR_DTE" && invoice.FiscalStatus != "REJECTED" {
		fields["fiscal_status"] = "El estado fiscal de la factura no permite preparar un DTE."
	}
	if invoice.Currency != "USD" {
		fields["currency"] = "La primera versión DTE de RentStage admite únicamente USD."
	}
	if len(invoice.Items) == 0 {
		fields["items"] = "La factura no contiene líneas."
	}
	if strings.TrimSpace(invoice.SellerLegalName) == "" {
		fields["seller_legal_name"] = "Completa la razón social del emisor."
	}
	if strings.TrimSpace(invoice.SellerTaxID) == "" {
		fields["seller_tax_id"] = "Completa el NIT del emisor."
	}
	if strings.TrimSpace(invoice.SellerRegistrationNumber) == "" {
		fields["seller_registration_number"] = "Completa el NRC del emisor."
	}
	if strings.TrimSpace(invoice.SellerEconomicCode) == "" || strings.TrimSpace(invoice.SellerEconomicActivity) == "" {
		fields["seller_economic_activity"] = "Completa el código y descripción de actividad económica del emisor."
	}
	if strings.TrimSpace(invoice.SellerAddress) == "" {
		fields["seller_address"] = "Completa la dirección fiscal del emisor."
	}
	if strings.TrimSpace(invoice.SellerDepartmentCode) == "" || strings.TrimSpace(invoice.SellerMunicipalityCode) == "" {
		fields["seller_location_codes"] = "Completa los códigos de departamento y municipio del emisor."
	}
	if strings.TrimSpace(invoice.SellerEmail) == "" || strings.TrimSpace(invoice.SellerPhone) == "" {
		fields["seller_contact"] = "Completa correo y teléfono del emisor."
	}
	if strings.TrimSpace(invoice.CustomerName) == "" {
		fields["customer_name"] = "La factura no contiene nombre del receptor."
	}
	if documentType == "01" && invoice.TotalAmount > 1095 {
		if strings.TrimSpace(invoice.CustomerTaxID) == "" {
			fields["customer_tax_id"] = "Las facturas superiores a $1,095 requieren identificación del receptor."
		}
		if strings.TrimSpace(invoice.CustomerAddress) == "" || strings.TrimSpace(invoice.CustomerDepartmentCode) == "" || strings.TrimSpace(invoice.CustomerMunicipalityCode) == "" {
			fields["customer_address"] = "Las facturas superiores a $1,095 requieren dirección y códigos geográficos del receptor."
		}
	}
	if documentType == "03" {
		if strings.TrimSpace(invoice.CustomerTaxID) == "" {
			fields["customer_tax_id"] = "El CCF requiere NIT del receptor."
		}
		if strings.TrimSpace(invoice.CustomerRegistrationNumber) == "" {
			fields["customer_registration_number"] = "El CCF requiere NRC del receptor."
		}
		if strings.TrimSpace(invoice.CustomerEconomicCode) == "" || strings.TrimSpace(invoice.CustomerEconomicActivity) == "" {
			fields["customer_economic_activity"] = "El CCF requiere actividad económica del receptor."
		}
		if strings.TrimSpace(invoice.CustomerAddress) == "" || strings.TrimSpace(invoice.CustomerDepartmentCode) == "" || strings.TrimSpace(invoice.CustomerMunicipalityCode) == "" {
			fields["customer_address"] = "El CCF requiere dirección y códigos geográficos del receptor."
		}
	}
	if settings.ProviderMode == "MH_HTTP" {
		for key, value := range map[string]string{
			"auth_url":                    settings.AuthURL,
			"signer_url":                  settings.SignerURL,
			"reception_url":               settings.ReceptionURL,
			"user_secret_ref":             settings.UserSecretRef,
			"password_secret_ref":         settings.PasswordSecretRef,
			"signing_password_secret_ref": settings.SigningPasswordSecretRef,
		} {
			if strings.TrimSpace(value) == "" {
				fields[key] = "Este valor es obligatorio para MH_HTTP."
			}
		}
	}
	return fields
}

func buildPayload(settings Settings, invoice InvoiceSnapshot, documentType, generationCode, control string) map[string]any {
	location := time.FixedZone("America/El_Salvador", -6*60*60)
	now := time.Now().In(location)
	issueDate := invoice.IssueDate
	if issueDate == "" {
		issueDate = now.Format("2006-01-02")
	}

	body := make([]map[string]any, 0, len(invoice.Items))
	for index, item := range invoice.Items {
		nonTaxable := 0.0
		exempt := 0.0
		taxable := 0.0
		switch item.TaxCategory {
		case "NON_TAXABLE":
			nonTaxable = round2(item.NetAmount)
		case "EXEMPT":
			exempt = round2(item.NetAmount)
		default:
			taxable = round2(item.NetAmount)
		}
		var tributes any
		if item.TaxCategory == "TAXABLE" {
			tributes = []string{"20"}
		}
		body = append(body, map[string]any{
			"numItem":         index + 1,
			"tipoItem":        item.DTEItemType,
			"numeroDocumento": nil,
			"codigo":          nullableString(item.DTEProductCode),
			"codTributo":      nil,
			"descripcion":     item.Description,
			"cantidad":        round3(item.Quantity),
			"uniMedida":       item.DTEUnitCode,
			"precioUni":       round4(item.UnitPrice),
			"montoDescu":      round2(item.DiscountAmount),
			"ventaNoSuj":      nonTaxable,
			"ventaExenta":     exempt,
			"ventaGravada":    taxable,
			"tributos":        tributes,
			"psv":             0.0,
			"noGravado":       0.0,
			"ivaItem":         round2(item.TaxAmount),
		})
	}

	condition := 1
	if invoice.DueDate > invoice.IssueDate {
		condition = 2
	}

	issuer := map[string]any{
		"nit":                 compactID(invoice.SellerTaxID),
		"nrc":                 compactID(invoice.SellerRegistrationNumber),
		"nombre":              invoice.SellerLegalName,
		"codActividad":        invoice.SellerEconomicCode,
		"descActividad":       invoice.SellerEconomicActivity,
		"nombreComercial":     nullableString(invoice.SellerTradeName),
		"tipoEstablecimiento": settings.EstablishmentType,
		"direccion": map[string]any{
			"departamento": invoice.SellerDepartmentCode,
			"municipio":    invoice.SellerMunicipalityCode,
			"complemento":  invoice.SellerAddress,
		},
		"telefono":        invoice.SellerPhone,
		"correo":          invoice.SellerEmail,
		"codEstableMH":    nullableString(settings.EstablishmentCode),
		"codEstable":      nullableString(settings.EstablishmentCode),
		"codPuntoVentaMH": nullableString(settings.PointOfSaleCode),
		"codPuntoVenta":   nullableString(settings.PointOfSaleCode),
	}

	var receiver any
	if documentType == "03" {
		receiver = map[string]any{
			"nit":             compactID(invoice.CustomerTaxID),
			"nrc":             compactID(invoice.CustomerRegistrationNumber),
			"nombre":          invoice.CustomerName,
			"codActividad":    invoice.CustomerEconomicCode,
			"descActividad":   invoice.CustomerEconomicActivity,
			"nombreComercial": nullableString(invoice.CustomerTradeName),
			"direccion": map[string]any{
				"departamento": invoice.CustomerDepartmentCode,
				"municipio":    invoice.CustomerMunicipalityCode,
				"complemento":  invoice.CustomerAddress,
			},
			"telefono": nullableString(invoice.CustomerPhone),
			"correo":   nullableString(invoice.CustomerEmail),
		}
	} else {
		receiver = map[string]any{
			"tipoDocumento": defaultString(invoice.CustomerDocumentType, "36"),
			"numDocumento":  nullableString(compactID(invoice.CustomerTaxID)),
			"nrc":           nullableString(compactID(invoice.CustomerRegistrationNumber)),
			"nombre":        invoice.CustomerName,
			"codActividad":  nullableString(invoice.CustomerEconomicCode),
			"descActividad": nullableString(invoice.CustomerEconomicActivity),
			"direccion":     optionalAddress(invoice),
			"telefono":      nullableString(invoice.CustomerPhone),
			"correo":        nullableString(invoice.CustomerEmail),
		}
	}

	tributes := make([]map[string]any, 0, 1)
	if invoice.TaxAmount > 0 {
		tributes = append(tributes, map[string]any{
			"codigo":      "20",
			"descripcion": "Impuesto al Valor Agregado 13%",
			"valor":       round2(invoice.TaxAmount),
		})
	}
	payments := []map[string]any{{
		"codigo":     "01",
		"montoPago":  round2(invoice.TotalAmount),
		"referencia": nil,
		"plazo":      nil,
		"periodo":    nil,
	}}

	return map[string]any{
		"identificacion": map[string]any{
			"version":          settings.SchemaVersion,
			"ambiente":         environmentCode(settings.Environment),
			"tipoDte":          documentType,
			"numeroControl":    control,
			"codigoGeneracion": strings.ToUpper(generationCode),
			"tipoModelo":       1,
			"tipoOperacion":    1,
			"tipoContingencia": nil,
			"motivoContin":     nil,
			"fecEmi":           issueDate,
			"horEmi":           now.Format("15:04:05"),
			"tipoMoneda":       invoice.Currency,
		},
		"documentoRelacionado": nil,
		"emisor":               issuer,
		"receptor":             receiver,
		"otrosDocumentos":      nil,
		"ventaTercero":         nil,
		"cuerpoDocumento":      body,
		"resumen": map[string]any{
			"totalNoSuj":          round2(invoice.NonTaxableAmount),
			"totalExenta":         round2(invoice.ExemptAmount),
			"totalGravada":        round2(invoice.TaxableAmount),
			"subTotalVentas":      round2(invoice.NonTaxableAmount + invoice.ExemptAmount + invoice.TaxableAmount),
			"descuNoSuj":          0.0,
			"descuExenta":         0.0,
			"descuGravada":        0.0,
			"porcentajeDescuento": 0.0,
			"totalDescu":          0.0,
			"tributos":            tributes,
			"subTotal":            round2(invoice.NonTaxableAmount + invoice.ExemptAmount + invoice.TaxableAmount),
			"ivaRete1":            0.0,
			"reteRenta":           0.0,
			"montoTotalOperacion": round2(invoice.TotalAmount),
			"totalNoGravado":      0.0,
			"totalPagar":          round2(invoice.TotalAmount),
			"totalLetras":         moneyWords(invoice.TotalAmount),
			"totalIva":            round2(invoice.TaxAmount),
			"saldoFavor":          0.0,
			"condicionOperacion":  condition,
			"pagos":               payments,
			"numPagoElectronico":  nil,
		},
		"extension": map[string]any{
			"nombEntrega":   nil,
			"docuEntrega":   nil,
			"nombRecibe":    nil,
			"docuRecibe":    nil,
			"observaciones": nullableString(strings.TrimSpace(invoice.Notes + " " + invoice.Terms)),
			"placaVehiculo": nil,
		},
		"apendice": []map[string]any{
			{"campo": "RentStageInvoice", "etiqueta": "Factura interna", "valor": fmt.Sprintf("%s-%06d", invoice.InvoicePrefix, invoice.InvoiceNumber)},
		},
	}
}

func optionalAddress(invoice InvoiceSnapshot) any {
	if strings.TrimSpace(invoice.CustomerAddress) == "" {
		return nil
	}
	return map[string]any{
		"departamento": defaultString(invoice.CustomerDepartmentCode, "00"),
		"municipio":    defaultString(invoice.CustomerMunicipalityCode, "00"),
		"complemento":  invoice.CustomerAddress,
	}
}

func compactID(value string) string {
	replacer := strings.NewReplacer("-", "", " ", "", ".", "")
	return replacer.Replace(strings.TrimSpace(value))
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func round2(value float64) float64 { return math.Round(value*100) / 100 }
func round3(value float64) float64 { return math.Round(value*1000) / 1000 }
func round4(value float64) float64 { return math.Round(value*10000) / 10000 }

func moneyWords(value float64) string {
	cents := int64(math.Round(value * 100))
	whole := cents / 100
	fraction := cents % 100
	return strings.ToUpper(fmt.Sprintf("%s DÓLARES DE LOS ESTADOS UNIDOS DE AMÉRICA CON %02d/100", numberWords(whole), fraction))
}

func numberWords(value int64) string {
	if value == 0 {
		return "cero"
	}
	if value < 0 {
		return "menos " + numberWords(-value)
	}
	parts := make([]string, 0)
	if value >= 1_000_000 {
		millions := value / 1_000_000
		if millions == 1 {
			parts = append(parts, "un millón")
		} else {
			parts = append(parts, numberWords(millions)+" millones")
		}
		value %= 1_000_000
	}
	if value >= 1000 {
		thousands := value / 1000
		if thousands == 1 {
			parts = append(parts, "mil")
		} else {
			parts = append(parts, numberWords(thousands)+" mil")
		}
		value %= 1000
	}
	if value > 0 {
		parts = append(parts, wordsBelowThousand(int(value)))
	}
	return strings.Join(parts, " ")
}

func wordsBelowThousand(value int) string {
	units := []string{"", "uno", "dos", "tres", "cuatro", "cinco", "seis", "siete", "ocho", "nueve", "diez", "once", "doce", "trece", "catorce", "quince", "dieciséis", "diecisiete", "dieciocho", "diecinueve", "veinte", "veintiuno", "veintidós", "veintitrés", "veinticuatro", "veinticinco", "veintiséis", "veintisiete", "veintiocho", "veintinueve"}
	tens := []string{"", "", "treinta", "cuarenta", "cincuenta", "sesenta", "setenta", "ochenta", "noventa"}
	hundreds := []string{"", "ciento", "doscientos", "trescientos", "cuatrocientos", "quinientos", "seiscientos", "setecientos", "ochocientos", "novecientos"}
	parts := make([]string, 0, 2)
	if value == 100 {
		return "cien"
	}
	if value >= 100 {
		parts = append(parts, hundreds[value/100])
		value %= 100
	}
	if value < 30 {
		if value > 0 {
			parts = append(parts, units[value])
		}
	} else {
		word := tens[value/10]
		if value%10 != 0 {
			word += " y " + units[value%10]
		}
		parts = append(parts, word)
	}
	return strings.Join(parts, " ")
}

func sequenceFromControl(control string) int64 {
	parts := strings.Split(control, "-")
	if len(parts) == 0 {
		return 0
	}
	value, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	return value
}
