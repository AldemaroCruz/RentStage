package quoteportal

import (
	"strings"
	"testing"
)

func TestTokenRoundTrip(t *testing.T) {
	token, tokenHash, err := newToken()
	if err != nil {
		t.Fatalf("newToken() error = %v", err)
	}
	if token == "" || tokenHash == "" {
		t.Fatal("token and hash must not be empty")
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("token is not raw URL-safe base64: %q", token)
	}
	if len(tokenHash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(tokenHash))
	}
	got, err := hashToken(token)
	if err != nil {
		t.Fatalf("hashToken() error = %v", err)
	}
	if got != tokenHash {
		t.Fatalf("hashToken() = %q, want %q", got, tokenHash)
	}
}

func TestHashTokenRejectsMalformedValues(t *testing.T) {
	for _, token := range []string{"", "short", "not+url/safe", strings.Repeat("a", 80)} {
		if _, err := hashToken(token); err == nil {
			t.Fatalf("hashToken(%q) should fail", token)
		}
	}
}

func TestNormalizeSettings(t *testing.T) {
	item, fields := normalizeSettings(SettingsInput{
		Enabled:                true,
		Headline:               "  Tu propuesta está lista  ",
		Introduction:           "  Revisa los detalles.  ",
		AccentColor:            "#AABBCC",
		DefaultValidityDays:    14,
		AllowRejection:         true,
		RequireResponseName:    true,
		AcceptanceTermsText:    "  Acepto las condiciones.  ",
		AcceptanceTermsVersion: " 2.0 ",
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected fields: %+v", fields)
	}
	if item.Headline != "Tu propuesta está lista" || item.Introduction != "Revisa los detalles." {
		t.Fatalf("text was not trimmed: %+v", item)
	}
	if item.AccentColor != "#aabbcc" {
		t.Fatalf("accent color = %q, want lowercase hex", item.AccentColor)
	}
	if item.AcceptanceTermsText != "Acepto las condiciones." || item.AcceptanceTermsVersion != "2.0" {
		t.Fatalf("terms were not normalized: %+v", item)
	}
}

func TestNormalizeSettingsRejectsInvalidValues(t *testing.T) {
	_, fields := normalizeSettings(SettingsInput{
		AccentColor:            "purple",
		DefaultValidityDays:    0,
		AcceptanceTermsVersion: "",
	})
	for _, key := range []string{"headline", "accent_color", "default_validity_days", "acceptance_terms_text", "acceptance_terms_version"} {
		if fields[key] == "" {
			t.Fatalf("expected validation field %q: %+v", key, fields)
		}
	}
}

func TestNormalizeAcceptRequiresTermsAndValidatesEmail(t *testing.T) {
	_, fields := normalizeAccept(AcceptInput{
		ResponseName:  " Ana López ",
		ResponseEmail: "Ana <ana@example.com>",
		TermsAccepted: false,
	})
	if fields["terms_accepted"] == "" || fields["response_email"] == "" {
		t.Fatalf("expected terms and email validation: %+v", fields)
	}

	decision, fields := normalizeAccept(AcceptInput{
		ResponseName:  " Ana López ",
		ResponseEmail: " ANA@EXAMPLE.COM ",
		TermsAccepted: true,
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected fields: %+v", fields)
	}
	if decision.ResponseName != "Ana López" || decision.ResponseEmail == nil || *decision.ResponseEmail != "ana@example.com" {
		t.Fatalf("unexpected normalized decision: %+v", decision)
	}
}

func TestNormalizeRejectTrimsOptionalReason(t *testing.T) {
	decision, fields := normalizeReject(RejectInput{
		ResponseName:    " Carlos ",
		RejectionReason: "  Necesito otra fecha.  ",
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected fields: %+v", fields)
	}
	if decision.ResponseName != "Carlos" || decision.RejectionReason == nil || *decision.RejectionReason != "Necesito otra fecha." {
		t.Fatalf("unexpected rejection decision: %+v", decision)
	}
}

func TestOriginHashIsStableAndTokenScoped(t *testing.T) {
	service := &Service{fingerprintSalt: "test-secret"}
	first := service.originHash("token-a", "127.0.0.1")
	second := service.originHash("token-a", "127.0.0.1")
	otherToken := service.originHash("token-b", "127.0.0.1")
	otherOrigin := service.originHash("token-a", "127.0.0.2")
	if first == "" || first != second {
		t.Fatalf("origin hash must be stable: %q %q", first, second)
	}
	if first == otherToken || first == otherOrigin {
		t.Fatal("origin hash must vary by portal token and origin")
	}
}

func TestCleanUserAgentUsesUnicodeSafeLimit(t *testing.T) {
	input := strings.Repeat("á", 510)
	got := cleanUserAgent(input)
	if len([]rune(got)) != 500 {
		t.Fatalf("user agent rune length = %d, want 500", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "á") {
		t.Fatal("user agent truncation split a UTF-8 rune")
	}
}
