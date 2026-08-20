package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                       string
	HTTPAddr                     string
	DatabaseURL                  string
	SeedDemoData                 bool
	AllowDemoDataOutsideLocal    bool
	CORSAllowedOrigins           []string
	FirebaseProjectID            string
	SessionCookieName            string
	TenantCookieName             string
	CSRFCookieName               string
	CookieSecure                 bool
	SessionDuration              time.Duration
	RequireVerifiedEmail         bool
	WebBaseURL                   string
	LocalAuthBootstrap           bool
	LocalOwnerEmail              string
	LocalOwnerPassword           string
	LocalOwnerDisplayName        string
	LocalDefaultTenantID         string
	PublicRequestFingerprintSalt string
	MetaWhatsAppMode             string
	MetaGraphBaseURL             string
	MetaGraphAPIVersion          string
	MetaPhoneNumberID            string
	MetaWABAID                   string
	MetaAccessToken              string
	MetaAppSecret                string
	MetaWebhookVerifyToken       string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:                       strings.ToLower(env("APP_ENV", "local")),
		HTTPAddr:                     resolveHTTPAddr(),
		DatabaseURL:                  strings.TrimSpace(os.Getenv("DATABASE_URL")),
		SeedDemoData:                 envBool("SEED_DEMO_DATA", false),
		AllowDemoDataOutsideLocal:    envBool("ALLOW_DEMO_DATA_OUTSIDE_LOCAL", false),
		FirebaseProjectID:            env("FIREBASE_PROJECT_ID", "demo-rentstage"),
		SessionCookieName:            env("SESSION_COOKIE_NAME", "rentstage_session"),
		TenantCookieName:             env("TENANT_COOKIE_NAME", "rentstage_tenant"),
		CSRFCookieName:               env("CSRF_COOKIE_NAME", "rentstage_csrf"),
		CookieSecure:                 envBool("COOKIE_SECURE", false),
		RequireVerifiedEmail:         envBool("REQUIRE_VERIFIED_EMAIL", false),
		WebBaseURL:                   strings.TrimRight(env("WEB_BASE_URL", "http://127.0.0.1:3000"), "/"),
		LocalAuthBootstrap:           envBool("LOCAL_AUTH_BOOTSTRAP", false),
		LocalOwnerEmail:              strings.ToLower(env("LOCAL_OWNER_EMAIL", "owner@rentstage.local")),
		LocalOwnerPassword:           env("LOCAL_OWNER_PASSWORD", "RentStage123!"),
		LocalOwnerDisplayName:        env("LOCAL_OWNER_DISPLAY_NAME", "Administrador Demo"),
		LocalDefaultTenantID:         env("LOCAL_DEFAULT_TENANT_ID", "00000000-0000-0000-0000-000000000001"),
		PublicRequestFingerprintSalt: env("PUBLIC_REQUEST_FINGERPRINT_SALT", "rentstage-local-public-catalog"),
		MetaWhatsAppMode:             strings.ToLower(env("META_WHATSAPP_MODE", "disabled")),
		MetaGraphBaseURL:             strings.TrimRight(env("META_GRAPH_BASE_URL", "https://graph.facebook.com"), "/"),
		MetaGraphAPIVersion:          env("META_GRAPH_API_VERSION", "v-test"),
		MetaPhoneNumberID:            strings.TrimSpace(os.Getenv("META_PHONE_NUMBER_ID")),
		MetaWABAID:                   strings.TrimSpace(os.Getenv("META_WABA_ID")),
		MetaAccessToken:              strings.TrimSpace(os.Getenv("META_ACCESS_TOKEN")),
		MetaAppSecret:                strings.TrimSpace(os.Getenv("META_APP_SECRET")),
		MetaWebhookVerifyToken:       strings.TrimSpace(os.Getenv("META_WEBHOOK_VERIFY_TOKEN")),
	}

	if cfg.AppEnv != "local" && cfg.AppEnv != "staging" && cfg.AppEnv != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be local, staging, or production")
	}

	origins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if origins != "" {
		for _, origin := range strings.Split(origins, ",") {
			if value := strings.TrimSpace(origin); value != "" {
				cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, strings.TrimRight(value, "/"))
			}
		}
	}

	durationValue := env("SESSION_DURATION", "12h")
	duration, err := time.ParseDuration(durationValue)
	if err != nil {
		return Config{}, fmt.Errorf("SESSION_DURATION is invalid: %w", err)
	}
	if duration < 5*time.Minute || duration > 14*24*time.Hour {
		return Config{}, fmt.Errorf("SESSION_DURATION must be between 5 minutes and 14 days")
	}
	cfg.SessionDuration = duration

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.FirebaseProjectID == "" {
		return Config{}, fmt.Errorf("FIREBASE_PROJECT_ID is required")
	}
	if cfg.LocalAuthBootstrap && (cfg.LocalOwnerEmail == "" || len(cfg.LocalOwnerPassword) < 6) {
		return Config{}, fmt.Errorf("local auth bootstrap requires LOCAL_OWNER_EMAIL and a password of at least 6 characters")
	}
	if err := validateMetaWhatsApp(cfg); err != nil {
		return Config{}, err
	}

	nonLocal := cfg.AppEnv != "local"
	if nonLocal && cfg.FirebaseProjectID == "demo-rentstage" {
		return Config{}, fmt.Errorf("FIREBASE_PROJECT_ID must reference a real project outside local development")
	}
	if nonLocal &&
		(cfg.PublicRequestFingerprintSalt == "rentstage-local-public-catalog" || len(cfg.PublicRequestFingerprintSalt) < 32) {
		return Config{}, fmt.Errorf("PUBLIC_REQUEST_FINGERPRINT_SALT must be an explicit random value of at least 32 characters outside local development")
	}
	if nonLocal && !cfg.CookieSecure {
		return Config{}, fmt.Errorf("COOKIE_SECURE must be true outside local development")
	}
	if nonLocal && cfg.LocalAuthBootstrap {
		return Config{}, fmt.Errorf("LOCAL_AUTH_BOOTSTRAP must be false outside local development")
	}
	if nonLocal && strings.TrimSpace(os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")) != "" {
		return Config{}, fmt.Errorf("FIREBASE_AUTH_EMULATOR_HOST must not be set outside local development")
	}
	if cfg.AppEnv == "production" && cfg.SeedDemoData {
		return Config{}, fmt.Errorf("SEED_DEMO_DATA must be false in production")
	}
	if cfg.AppEnv == "staging" && cfg.SeedDemoData && !cfg.AllowDemoDataOutsideLocal {
		return Config{}, fmt.Errorf("SEED_DEMO_DATA in staging requires ALLOW_DEMO_DATA_OUTSIDE_LOCAL=true")
	}
	if nonLocal {
		if err := validateHTTPSOrigin(cfg.WebBaseURL, "WEB_BASE_URL"); err != nil {
			return Config{}, err
		}
		if len(cfg.CORSAllowedOrigins) == 0 {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain WEB_BASE_URL outside local development")
		}
		webOriginPresent := false
		for _, origin := range cfg.CORSAllowedOrigins {
			if err := validateHTTPSOrigin(origin, "CORS_ALLOWED_ORIGINS"); err != nil {
				return Config{}, err
			}
			if origin == cfg.WebBaseURL {
				webOriginPresent = true
			}
		}
		if !webOriginPresent {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain WEB_BASE_URL outside local development")
		}
	}
	if cfg.AppEnv == "production" && !cfg.RequireVerifiedEmail {
		return Config{}, fmt.Errorf("REQUIRE_VERIFIED_EMAIL must be true in production")
	}

	return cfg, nil
}

func validateMetaWhatsApp(cfg Config) error {
	switch cfg.MetaWhatsAppMode {
	case "disabled":
		return nil
	case "local_mock":
		if cfg.AppEnv != "local" {
			return fmt.Errorf("META_WHATSAPP_MODE=local_mock is allowed only in local development")
		}
		if err := validateLocalMetaGraphURL(cfg.MetaGraphBaseURL); err != nil {
			return err
		}
	case "cloud":
		if err := validateHTTPSOrigin(cfg.MetaGraphBaseURL, "META_GRAPH_BASE_URL"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("META_WHATSAPP_MODE must be disabled, local_mock, or cloud")
	}

	required := []struct {
		name  string
		value string
	}{
		{name: "META_PHONE_NUMBER_ID", value: cfg.MetaPhoneNumberID},
		{name: "META_WABA_ID", value: cfg.MetaWABAID},
		{name: "META_GRAPH_API_VERSION", value: cfg.MetaGraphAPIVersion},
		{name: "META_ACCESS_TOKEN", value: cfg.MetaAccessToken},
		{name: "META_APP_SECRET", value: cfg.MetaAppSecret},
		{name: "META_WEBHOOK_VERIFY_TOKEN", value: cfg.MetaWebhookVerifyToken},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required when Meta WhatsApp is enabled", field.name)
		}
	}
	if len(cfg.MetaAppSecret) < 16 || len(cfg.MetaWebhookVerifyToken) < 16 {
		return fmt.Errorf("META_APP_SECRET and META_WEBHOOK_VERIFY_TOKEN must contain at least 16 characters")
	}
	return nil
}

func validateLocalMetaGraphURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("META_GRAPH_BASE_URL in local_mock mode must use the local HTTP harness")
	}
	hostname := parsed.Hostname()
	if hostname != "127.0.0.1" && hostname != "localhost" && hostname != "::1" {
		return fmt.Errorf("META_GRAPH_BASE_URL in local_mock mode must use a loopback host")
	}
	if !strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/api/v1/integrations/meta/local-graph") {
		return fmt.Errorf("META_GRAPH_BASE_URL in local_mock mode must target the local Graph harness")
	}
	return nil
}

func validateHTTPSOrigin(value, field string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || strings.Contains(parsed.Host, "*") {
		return fmt.Errorf("%s must contain HTTPS origins outside local development", field)
	}
	if parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s entries must be origins without paths, credentials, queries, or fragments", field)
	}
	return nil
}

func resolveHTTPAddr() string {
	if value := strings.TrimSpace(os.Getenv("HTTP_ADDR")); value != "" {
		return value
	}
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return ":" + strings.TrimPrefix(port, ":")
	}
	return ":8080"
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
