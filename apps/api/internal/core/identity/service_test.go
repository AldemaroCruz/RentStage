package identity

import "testing"

func TestNormalizeOrganization(t *testing.T) {
	input, fields := NormalizeOrganization(CreateOrganizationInput{
		Name:     "  Audio Pró SV  ",
		Slug:     "",
		Email:    "VENTAS@EXAMPLE.COM",
		Currency: "usd",
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation errors: %#v", fields)
	}
	if input.Name != "Audio Pró SV" {
		t.Fatalf("name = %q", input.Name)
	}
	if input.Slug != "audio-pr-sv" {
		t.Fatalf("slug = %q, want audio-pr-sv", input.Slug)
	}
	if input.Email != "ventas@example.com" {
		t.Fatalf("email = %q", input.Email)
	}
	if input.CountryCode != "SV" || input.Currency != "USD" || input.Timezone != "America/El_Salvador" {
		t.Fatalf("defaults not applied: %#v", input)
	}
}

func TestNormalizeOrganizationRejectsInvalidFields(t *testing.T) {
	_, fields := NormalizeOrganization(CreateOrganizationInput{
		Name:        "",
		Slug:        "---",
		Email:       "not-an-email",
		CountryCode: "ELS",
		Currency:    "US",
		Timezone:    "",
	})
	for _, key := range []string{"name", "slug", "email", "country_code", "currency"} {
		if fields[key] == "" {
			t.Fatalf("expected validation error for %s: %#v", key, fields)
		}
	}
}
