package dte

import (
	"context"
	"strings"
	"testing"
)

func TestMockProviderLifecycle(t *testing.T) {
	invoice := completeInvoiceSnapshot()
	settings := completeSettings()
	control := controlNumber("01", settings.EstablishmentCode, settings.PointOfSaleCode, 1)
	document := DocumentDetail{
		DocumentSummary: DocumentSummary{
			DocumentType:   "01",
			SchemaVersion:  1,
			GenerationCode: "E5AA7B13-0675-4AFA-917A-8A9ACEDDA281",
			ControlNumber:  control,
		},
		Payload: buildPayload(settings, invoice, "01", "E5AA7B13-0675-4AFA-917A-8A9ACEDDA281", control),
	}

	provider := MockProvider{}
	result, err := provider.Submit(context.Background(), ProviderSubmission{Settings: settings, Document: document})
	if err != nil {
		t.Fatalf("mock submit returned error: %v", err)
	}
	if !result.Accepted || !strings.HasPrefix(result.ReceiptSeal, "MOCK-") || !strings.HasPrefix(result.SignedDocument, "MOCK-SIGNED.") {
		t.Fatalf("unexpected mock result: %#v", result)
	}

	document.ReceiptSeal = result.ReceiptSeal
	invalidation, err := provider.Invalidate(context.Background(), ProviderSubmission{Settings: settings, Document: document}, "Error en datos del receptor")
	if err != nil {
		t.Fatalf("mock invalidation returned error: %v", err)
	}
	if !invalidation.Accepted || invalidation.ProviderStatus != "INVALIDADO" {
		t.Fatalf("unexpected invalidation result: %#v", invalidation)
	}
}

func TestRedactMapRemovesCredentialValues(t *testing.T) {
	redacted := redactMap(map[string]any{
		"user":         "06140000000000",
		"pwd":          "secret-password",
		"access_token": "secret-token",
		"safe":         "visible",
		"nested": map[string]any{
			"signing_password": "nested-secret",
			"document":         "visible-document",
		},
	})
	if redacted["pwd"] != "***" || redacted["access_token"] != "***" {
		t.Fatalf("credentials were not redacted: %#v", redacted)
	}
	if redacted["safe"] != "visible" {
		t.Fatalf("safe value was unexpectedly changed")
	}
	nested, ok := redacted["nested"].(map[string]any)
	if !ok || nested["signing_password"] != "***" || nested["document"] != "visible-document" {
		t.Fatalf("nested credentials were not redacted: %#v", redacted)
	}
}

func TestProviderEndpointRejectsPrivateAndCredentialedTargets(t *testing.T) {
	for _, endpoint := range []string{
		"http://127.0.0.1/auth",
		"http://169.254.169.254/latest/meta-data",
		"https://user:password@example.com/auth",
		"ftp://example.com/auth",
	} {
		if err := validateProviderEndpoint(endpoint, false); err == nil {
			t.Fatalf("unsafe provider endpoint was accepted: %s", endpoint)
		}
	}
	if err := validateProviderEndpoint("https://example.com/auth", true); err != nil {
		t.Fatalf("public HTTPS endpoint was rejected: %v", err)
	}
}

func TestNewMHHTTPProviderRequiresHTTPSInProduction(t *testing.T) {
	settings := completeSettings()
	settings.ProviderMode = "MH_HTTP"
	settings.Environment = "PRODUCTION"
	settings.AuthURL = "http://example.invalid/auth"
	settings.SignerURL = "https://example.invalid/sign"
	settings.ReceptionURL = "https://example.invalid/receive"
	if _, err := NewMHHTTPProvider(settings); err == nil {
		t.Fatalf("production provider accepted a non-HTTPS endpoint")
	}
}

func TestNewMHHTTPProviderValidatesOptionalEndpoints(t *testing.T) {
	settings := completeSettings()
	settings.ProviderMode = "MH_HTTP"
	settings.Environment = "TEST"
	settings.AuthURL = "https://example.com/auth"
	settings.SignerURL = "https://example.com/sign"
	settings.ReceptionURL = "https://example.com/receive"
	settings.InvalidationURL = "http://127.0.0.1/invalidate"
	if _, err := NewMHHTTPProvider(settings); err == nil {
		t.Fatalf("provider accepted an unsafe optional invalidation endpoint")
	}
}

func TestNormalizeSettingsRejectsUnsafeProviderEndpoint(t *testing.T) {
	input := SettingsInput{
		Enabled:                  true,
		ProviderMode:             "MH_HTTP",
		Environment:              "TEST",
		DefaultDocumentType:      "01",
		SchemaVersion:            1,
		EstablishmentType:        "01",
		EstablishmentCode:        "M001",
		PointOfSaleCode:          "P001",
		AuthURL:                  "http://169.254.169.254/auth",
		SignerURL:                "https://example.com/sign",
		ReceptionURL:             "https://example.com/receive",
		UserSecretRef:            "env://DTE_MH_USER",
		PasswordSecretRef:        "env://DTE_MH_PASSWORD",
		SigningPasswordSecretRef: "env://DTE_MH_SIGNING_PASSWORD",
		MaxAttempts:              5,
		RetryBaseSeconds:         60,
	}
	_, fields := normalizeSettings(input)
	if fields["auth_url"] == "" {
		t.Fatalf("unsafe endpoint was not rejected during settings validation: %#v", fields)
	}
}

func TestSettingsForDocumentPreservesPreparedProviderIdentity(t *testing.T) {
	current := completeSettings()
	current.ProviderMode = "MH_HTTP"
	current.Environment = "PRODUCTION"
	current.AuthURL = "https://example.com/auth"
	current.SignerURL = "https://example.com/sign"
	current.ReceptionURL = "https://example.com/receive"
	current.UserSecretRef = "env://DTE_MH_USER"
	current.PasswordSecretRef = "env://DTE_MH_PASSWORD"
	current.SigningPasswordSecretRef = "env://DTE_MH_SIGNING_PASSWORD"

	prepared := DocumentSummary{ProviderMode: "MOCK", Environment: "TEST", SchemaVersion: 1}
	execution := settingsForDocument(current, prepared)
	if execution.ProviderMode != "MOCK" || execution.Environment != "TEST" {
		t.Fatalf("prepared provider identity was not preserved: %#v", execution)
	}
	if !execution.ConfigurationReady {
		t.Fatalf("prepared MOCK document should remain executable in TEST mode: %#v", execution)
	}
}

func TestSamePreparationSettingsDetectsFiscalIdentityChanges(t *testing.T) {
	expected := completeSettings()
	locked := expected
	if !samePreparationSettings(expected, locked) {
		t.Fatalf("identical preparation settings were considered different")
	}
	locked.PointOfSaleCode = "P002"
	if samePreparationSettings(expected, locked) {
		t.Fatalf("changed point-of-sale code was not detected")
	}
}

func TestRetryDelayIsBoundedExponentialBackoff(t *testing.T) {
	if got := retryDelay(60, 1).Seconds(); got != 60 {
		t.Fatalf("first retry delay = %v, want 60", got)
	}
	if got := retryDelay(60, 10).Seconds(); got != 3840 {
		t.Fatalf("bounded retry delay = %v, want 3840", got)
	}
}
