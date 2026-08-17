package customer

import "testing"

func TestNormalizeCustomer(t *testing.T) {
	phone := "+503 7123-4567"
	email := "ALDEMARO@EXAMPLE.COM"
	company := "  Audio Pro  "

	result, fields := normalize(CreateInput{
		FirstName:         "  Aldemaro ",
		LastName:          " Cruz ",
		Phone:             &phone,
		Email:             &email,
		CompanyName:       &company,
		PreferredLanguage: "ES",
		Source:            "whatsapp",
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if result.FirstName != "Aldemaro" || result.LastName != "Cruz" {
		t.Fatalf("unexpected normalized name: %#v", result)
	}
	if result.Phone == nil || *result.Phone != "+50371234567" {
		t.Fatalf("unexpected phone: %#v", result.Phone)
	}
	if result.Email == nil || *result.Email != "aldemaro@example.com" {
		t.Fatalf("unexpected email: %#v", result.Email)
	}
	if result.Source != "WHATSAPP" || result.PreferredLanguage != "es" {
		t.Fatalf("unexpected source/language: %#v", result)
	}
}

func TestNormalizeCustomerRejectsLocalPhone(t *testing.T) {
	phone := "7123-4567"
	_, fields := normalize(CreateInput{FirstName: "A", Phone: &phone})
	if fields["phone"] == "" {
		t.Fatal("expected E.164 validation error")
	}
}
