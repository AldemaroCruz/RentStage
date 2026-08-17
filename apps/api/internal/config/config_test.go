package config

import (
	"strings"
	"testing"
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
