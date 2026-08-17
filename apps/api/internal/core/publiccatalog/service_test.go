package publiccatalog

import (
	"testing"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
)

func TestNormalizeSettings(t *testing.T) {
	cover := " https://cdn.example.com/cover.jpg "
	email := " ventas@example.com "
	item, fields := normalizeSettings(SettingsInput{
		Enabled:              true,
		Headline:             "  Sonido para tu evento  ",
		Description:          "  Catálogo público  ",
		CoverImageURL:        &cover,
		AccentColor:          "#6a57f7",
		ShowPrices:           true,
		ShowResources:        true,
		QuoteRequestsEnabled: true,
		ContactEmail:         &email,
		TermsText:            "  Autorizo contacto.  ",
		TermsVersion:         " 2.1 ",
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %+v", fields)
	}
	if item.Headline != "Sonido para tu evento" || item.AccentColor != "#6A57F7" {
		t.Fatalf("settings were not normalized: %+v", item)
	}
	if item.CoverImageURL == nil || *item.CoverImageURL != "https://cdn.example.com/cover.jpg" {
		t.Fatalf("cover URL was not normalized: %+v", item.CoverImageURL)
	}
	if item.ContactEmail == nil || *item.ContactEmail != "ventas@example.com" {
		t.Fatalf("email was not normalized: %+v", item.ContactEmail)
	}
}

func TestNormalizeSettingsRejectsUnsafeValues(t *testing.T) {
	cover := "javascript:alert(1)"
	email := "not-an-email"
	_, fields := normalizeSettings(SettingsInput{
		CoverImageURL: &cover,
		AccentColor:   "purple",
		ContactEmail:  &email,
		TermsText:     "",
	})
	for _, key := range []string{"cover_image_url", "accent_color", "contact_email", "terms_text"} {
		if fields[key] == "" {
			t.Fatalf("expected validation error for %s: %+v", key, fields)
		}
	}
}

func TestNormalizeQuoteRequest(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	phone := " +503 7000-0000 "
	email := " CLIENTE@EXAMPLE.COM "
	result, fields := normalizeQuoteRequest(QuoteRequestInput{
		FirstName:         " Ana ",
		LastName:          " López ",
		Phone:             &phone,
		Email:             &email,
		PreferredLanguage: "ES",
		StartAt:           now.Add(24 * time.Hour).Format(time.RFC3339),
		EndAt:             now.Add(30 * time.Hour).Format(time.RFC3339),
		ConsentAccepted:   true,
		Selections: []QuoteRequestSelection{{
			PackageSlug: " Paquete-Fiesta-100-Personas ",
			Quantity:    2,
		}},
	}, now)
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %+v", fields)
	}
	if result.FirstName != "Ana" || result.Email == nil || *result.Email != "cliente@example.com" {
		t.Fatalf("request was not normalized: %+v", result)
	}
	if len(result.Selections) != 1 || result.Selections[0].PackageSlug != "paquete-fiesta-100-personas" {
		t.Fatalf("selections were not normalized: %+v", result.Selections)
	}
}

func TestNormalizeQuoteRequestRequiresContactConsentAndUniquePackages(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	_, fields := normalizeQuoteRequest(QuoteRequestInput{
		FirstName: "Ana",
		StartAt:   now.Add(time.Hour).Format(time.RFC3339),
		EndAt:     now.Add(2 * time.Hour).Format(time.RFC3339),
		Selections: []QuoteRequestSelection{
			{PackageSlug: "fiesta", Quantity: 1},
			{PackageSlug: "fiesta", Quantity: 1},
		},
	}, now)
	for _, key := range []string{"contact", "consent_accepted", "selections[1].package_slug"} {
		if fields[key] == "" {
			t.Fatalf("expected validation error for %s: %+v", key, fields)
		}
	}
}

func TestSanitizeAvailabilityHidesInternalInventoryDetails(t *testing.T) {
	input := availability.Result{
		StartAt:   time.Now(),
		EndAt:     time.Now().Add(time.Hour),
		Available: false,
		Items: []availability.ItemResult{{
			ResourceID:        "internal-resource-id",
			ResourceName:      "JBL PRX815W",
			RequestedQuantity: 2,
			AvailableQuantity: 1,
			CanFulfill:        false,
		}},
	}
	result := sanitizeAvailability(input)
	if len(result.Items) != 1 || result.Items[0].ResourceName != "JBL PRX815W" || result.Items[0].CanFulfill {
		t.Fatalf("unexpected sanitized result: %+v", result)
	}
}

func TestSubmitterHashIsTenantScoped(t *testing.T) {
	service := &Service{fingerprintSalt: "secret"}
	first := service.submitterHash("tenant-a", "127.0.0.1")
	second := service.submitterHash("tenant-a", "127.0.0.1")
	otherTenant := service.submitterHash("tenant-b", "127.0.0.1")
	if first == "" || first != second {
		t.Fatalf("hash must be stable: %q %q", first, second)
	}
	if first == otherTenant {
		t.Fatal("hash must be scoped by tenant")
	}
}

func TestSlugifyAndUnicodeTruncation(t *testing.T) {
	if got := slugify("Consola e Iluminación / VIP"); got != "consola-e-iluminacion-vip" {
		t.Fatalf("slugify() = %q", got)
	}
	if got := truncate("áéíóú", 3); got != "áéí" {
		t.Fatalf("truncate split unicode text: %q", got)
	}
}

func TestValidEmailRejectsDisplayNameSyntax(t *testing.T) {
	if !validEmail("customer@example.com") {
		t.Fatal("plain email should be valid")
	}
	if validEmail("Customer <customer@example.com>") {
		t.Fatal("display-name email syntax should not be persisted as a customer email")
	}
}
