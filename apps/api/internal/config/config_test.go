package config

import (
	"strings"
	"testing"
	"time"
)

func setValidLocalEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "local")
	t.Setenv("DATABASE_URL", "postgres://rentstage:password@localhost/rentstage?sslmode=disable")
	t.Setenv("FIREBASE_PROJECT_ID", "demo-rentstage")
	t.Setenv("PUBLIC_REQUEST_FINGERPRINT_SALT", "rentstage-local-public-catalog")
	t.Setenv("LOCAL_AUTH_BOOTSTRAP", "false")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("REQUIRE_VERIFIED_EMAIL", "false")
	t.Setenv("WEB_BASE_URL", "http://127.0.0.1:3000")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://127.0.0.1:3000")
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	t.Setenv("SEED_DEMO_DATA", "false")
	t.Setenv("ALLOW_DEMO_DATA_OUTSIDE_LOCAL", "false")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("PORT", "")
	t.Setenv("META_WHATSAPP_MODE", "disabled")
	t.Setenv("META_OUTBOUND_ENABLED", "false")
	t.Setenv("META_GRAPH_BASE_URL", "")
	t.Setenv("META_GRAPH_API_VERSION", "")
	t.Setenv("META_PHONE_NUMBER_ID", "")
	t.Setenv("META_WABA_ID", "")
	t.Setenv("META_ACCESS_TOKEN", "")
	t.Setenv("META_APP_SECRET", "")
	t.Setenv("META_WEBHOOK_VERIFY_TOKEN", "")
	t.Setenv("ASSISTANT_AI_MODE", "rules")
	t.Setenv("ASSISTANT_AI_PROJECT_ID", "")
	t.Setenv("ASSISTANT_AI_LOCATION", "")
	t.Setenv("ASSISTANT_AI_MODEL", "")
	t.Setenv("ASSISTANT_AI_TIMEOUT", "")
	t.Setenv("ASSISTANT_AI_MAX_OUTPUT_TOKENS", "")
}

func TestLoadUsesSafeAssistantAIDefaults(t *testing.T) {
	setValidLocalEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}

	if cfg.AssistantAIMode != "rules" {
		t.Fatalf(
			"AssistantAIMode = %q, want rules",
			cfg.AssistantAIMode,
		)
	}
	if cfg.AssistantAILocation != "us-central1" {
		t.Fatalf(
			"AssistantAILocation = %q",
			cfg.AssistantAILocation,
		)
	}
	if cfg.AssistantAIModel != "gemini-2.5-flash" {
		t.Fatalf(
			"AssistantAIModel = %q",
			cfg.AssistantAIModel,
		)
	}
	if cfg.AssistantAITimeout != 20*time.Second {
		t.Fatalf(
			"AssistantAITimeout = %s",
			cfg.AssistantAITimeout,
		)
	}
	if cfg.AssistantAIMaxOutputTokens != 512 {
		t.Fatalf(
			"AssistantAIMaxOutputTokens = %d",
			cfg.AssistantAIMaxOutputTokens,
		)
	}
}

func TestLoadAllowsVertexAssistantAIWithExplicitConfiguration(t *testing.T) {
	setValidLocalEnvironment(t)
	t.Setenv("ASSISTANT_AI_MODE", "vertex")
	t.Setenv("ASSISTANT_AI_PROJECT_ID", "rentstage-ai-test")
	t.Setenv("ASSISTANT_AI_LOCATION", "us-central1")
	t.Setenv("ASSISTANT_AI_MODEL", "gemini-2.5-flash")
	t.Setenv("ASSISTANT_AI_TIMEOUT", "6s")
	t.Setenv("ASSISTANT_AI_MAX_OUTPUT_TOKENS", "384")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if cfg.AssistantAIMode != "vertex" {
		t.Fatalf(
			"AssistantAIMode = %q, want vertex",
			cfg.AssistantAIMode,
		)
	}
	if cfg.AssistantAIProjectID != "rentstage-ai-test" {
		t.Fatalf(
			"AssistantAIProjectID = %q",
			cfg.AssistantAIProjectID,
		)
	}
}

func TestLoadRejectsUnknownAssistantAIMode(t *testing.T) {
	setValidLocalEnvironment(t)
	t.Setenv("ASSISTANT_AI_MODE", "automatic")

	_, err := Load()
	if err == nil ||
		!strings.Contains(err.Error(), "ASSISTANT_AI_MODE") {
		t.Fatalf(
			"expected assistant AI mode validation error, got %v",
			err,
		)
	}
}

func TestLoadRejectsVertexAssistantAIWithoutProject(t *testing.T) {
	setValidLocalEnvironment(t)
	t.Setenv("ASSISTANT_AI_MODE", "vertex")

	_, err := Load()
	if err == nil ||
		!strings.Contains(err.Error(), "ASSISTANT_AI_PROJECT_ID") {
		t.Fatalf(
			"expected assistant AI project validation error, got %v",
			err,
		)
	}
}

func TestLoadRejectsInvalidAssistantAILimits(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{
			name:  "timeout syntax",
			key:   "ASSISTANT_AI_TIMEOUT",
			value: "soon",
		},
		{
			name:  "timeout too long",
			key:   "ASSISTANT_AI_TIMEOUT",
			value: "30s",
		},
		{
			name:  "token syntax",
			key:   "ASSISTANT_AI_MAX_OUTPUT_TOKENS",
			value: "many",
		},
		{
			name:  "tokens too low",
			key:   "ASSISTANT_AI_MAX_OUTPUT_TOKENS",
			value: "32",
		},
		{
			name:  "tokens too high",
			key:   "ASSISTANT_AI_MAX_OUTPUT_TOKENS",
			value: "4096",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidLocalEnvironment(t)
			t.Setenv(test.key, test.value)

			if _, err := Load(); err == nil {
				t.Fatalf(
					"expected validation error for %s=%q",
					test.key,
					test.value,
				)
			}
		})
	}
}

func setValidStagingEnvironment(t *testing.T) {
	t.Helper()
	setValidLocalEnvironment(t)
	t.Setenv("APP_ENV", "staging")
	t.Setenv("FIREBASE_PROJECT_ID", "rentstage-staging-123")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("WEB_BASE_URL", "https://rentstage-staging.example.run.app")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://rentstage-staging.example.run.app")
	t.Setenv("PUBLIC_REQUEST_FINGERPRINT_SALT", strings.Repeat("a", 64))
}

func TestLoadUsesCloudRunPortWhenHTTPAddrIsAbsent(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9090")
	}
}

func TestLoadPrefersExplicitHTTPAddr(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("PORT", "9090")
	t.Setenv("HTTP_ADDR", "127.0.0.1:8088")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8088" {
		t.Fatalf("HTTPAddr = %q, want explicit value", cfg.HTTPAddr)
	}
}

func TestLoadRejectsUnknownEnvironment(t *testing.T) {
	setValidLocalEnvironment(t)
	t.Setenv("APP_ENV", "qa")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "APP_ENV") {
		t.Fatalf("expected APP_ENV validation error, got %v", err)
	}
}

func TestLoadRejectsLocalFirebaseProjectOutsideLocal(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("FIREBASE_PROJECT_ID", "demo-rentstage")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "real project") {
		t.Fatalf("expected Firebase project validation error, got %v", err)
	}
}

func TestLoadRejectsInsecureStagingCookies(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("COOKIE_SECURE", "false")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "COOKIE_SECURE") {
		t.Fatalf("expected COOKIE_SECURE validation error, got %v", err)
	}
}

func TestLoadRejectsLocalBootstrapOutsideLocal(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("LOCAL_AUTH_BOOTSTRAP", "true")
	t.Setenv("LOCAL_OWNER_PASSWORD", "safe-enough-password")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "LOCAL_AUTH_BOOTSTRAP") {
		t.Fatalf("expected LOCAL_AUTH_BOOTSTRAP validation error, got %v", err)
	}
}

func TestLoadRejectsFirebaseEmulatorOutsideLocal(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "localhost:9099")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "FIREBASE_AUTH_EMULATOR_HOST") {
		t.Fatalf("expected emulator validation error, got %v", err)
	}
}

func TestLoadRejectsDemoSeedInStagingWithoutExplicitOverride(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("SEED_DEMO_DATA", "true")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ALLOW_DEMO_DATA_OUTSIDE_LOCAL") {
		t.Fatalf("expected demo-data validation error, got %v", err)
	}
}

func TestLoadAllowsDemoSeedInStagingWithExplicitOverride(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("SEED_DEMO_DATA", "true")
	t.Setenv("ALLOW_DEMO_DATA_OUTSIDE_LOCAL", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if !cfg.SeedDemoData || !cfg.AllowDemoDataOutsideLocal {
		t.Fatal("expected demo seed and explicit override to be true")
	}
}

func TestLoadRejectsDemoSeedInProductionEvenWithOverride(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("REQUIRE_VERIFIED_EMAIL", "true")
	t.Setenv("SEED_DEMO_DATA", "true")
	t.Setenv("ALLOW_DEMO_DATA_OUTSIDE_LOCAL", "true")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SEED_DEMO_DATA") {
		t.Fatalf("expected production demo-data validation error, got %v", err)
	}
}

func TestLoadRejectsNonHTTPSWebBaseOutsideLocal(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("WEB_BASE_URL", "http://rentstage.example.com")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "WEB_BASE_URL") {
		t.Fatalf("expected WEB_BASE_URL validation error, got %v", err)
	}
}

func TestLoadAllowsLocalMetaMockWithExplicitCredentials(t *testing.T) {
	setValidLocalEnvironment(t)
	t.Setenv("META_WHATSAPP_MODE", "local_mock")
	t.Setenv("META_OUTBOUND_ENABLED", "true")
	t.Setenv("META_GRAPH_BASE_URL", "http://127.0.0.1:8080/api/v1/integrations/meta/local-graph")
	t.Setenv("META_GRAPH_API_VERSION", "v-test")
	t.Setenv("META_PHONE_NUMBER_ID", "100000000000001")
	t.Setenv("META_WABA_ID", "200000000000001")
	t.Setenv("META_ACCESS_TOKEN", "rentstage-local-meta-access-token")
	t.Setenv("META_APP_SECRET", "rentstage-local-meta-app-secret")
	t.Setenv("META_WEBHOOK_VERIFY_TOKEN", "rentstage-local-meta-verify-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if cfg.MetaWhatsAppMode != "local_mock" {
		t.Fatalf("MetaWhatsAppMode = %q", cfg.MetaWhatsAppMode)
	}
	if !cfg.MetaOutboundEnabled {
		t.Fatal("expected local outbound harness to be enabled")
	}
}

func TestLoadRejectsCloudMetaOutboundInApplicationReadinessRelease(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("META_WHATSAPP_MODE", "cloud")
	t.Setenv("META_OUTBOUND_ENABLED", "true")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "real delivery remains deferred") {
		t.Fatalf("expected cloud outbound safety error, got %v", err)
	}
}

func TestLoadRejectsLocalMetaMockOutsideLocal(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("META_WHATSAPP_MODE", "local_mock")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "only in local") {
		t.Fatalf("expected local Meta mock isolation error, got %v", err)
	}
}

func TestLoadRejectsLocalMetaMockWithRemoteGraphEndpoint(t *testing.T) {
	setValidLocalEnvironment(t)
	t.Setenv("META_WHATSAPP_MODE", "local_mock")
	t.Setenv("META_GRAPH_BASE_URL", "http://graph.facebook.com/api/v1/integrations/meta/local-graph")
	t.Setenv("META_PHONE_NUMBER_ID", "100000000000001")
	t.Setenv("META_WABA_ID", "200000000000001")
	t.Setenv("META_ACCESS_TOKEN", "rentstage-local-meta-access-token")
	t.Setenv("META_APP_SECRET", "rentstage-local-meta-app-secret")
	t.Setenv("META_WEBHOOK_VERIFY_TOKEN", "rentstage-local-meta-verify-token")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected remote local_mock endpoint to be rejected, got %v", err)
	}
}

func TestLoadRejectsCloudMetaWithoutCredentials(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("META_WHATSAPP_MODE", "cloud")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "META_PHONE_NUMBER_ID") {
		t.Fatalf("expected missing Meta credential error, got %v", err)
	}
}

func TestLoadRejectsMissingOrMismatchedCORSOriginsOutsideLocal(t *testing.T) {
	for _, value := range []string{"", "https://different.example.com"} {
		t.Run(value, func(t *testing.T) {
			setValidStagingEnvironment(t)
			t.Setenv("CORS_ALLOWED_ORIGINS", value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
				t.Fatalf("expected CORS validation error for %q, got %v", value, err)
			}
		})
	}
}

func TestLoadRejectsCORSWildcardPathQueryOrCredentialsOutsideLocal(t *testing.T) {
	for _, value := range []string{
		"*",
		"https://*.example.com",
		"https://rentstage.example.com/path",
		"https://rentstage.example.com?debug=1",
		"https://user:password@rentstage.example.com",
	} {
		t.Run(value, func(t *testing.T) {
			setValidStagingEnvironment(t)
			t.Setenv("CORS_ALLOWED_ORIGINS", value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
				t.Fatalf("expected CORS validation error for %q, got %v", value, err)
			}
		})
	}
}

func TestLoadRequiresVerifiedEmailInProduction(t *testing.T) {
	setValidStagingEnvironment(t)
	t.Setenv("APP_ENV", "production")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "REQUIRE_VERIFIED_EMAIL") {
		t.Fatalf("expected production verified-email validation error, got %v", err)
	}
}
