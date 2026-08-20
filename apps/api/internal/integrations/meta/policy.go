package meta

import (
	"strings"
	"unicode"
)

type ConsentDecision string

const (
	ConsentUnchanged ConsentDecision = "UNCHANGED"
	ConsentOptedIn   ConsentDecision = "OPTED_IN"
	ConsentOptedOut  ConsentDecision = "OPTED_OUT"
)

var optOutPhrases = map[string]struct{}{
	"stop":             {},
	"salir":            {},
	"cancelar":         {},
	"baja":             {},
	"no mas mensajes":  {},
	"dejar de recibir": {},
}

var optInPhrases = map[string]struct{}{
	"start":     {},
	"iniciar":   {},
	"continuar": {},
	"alta":      {},
}

func ClassifyConsent(text string) ConsentDecision {
	normalized := normalizeConsentText(text)
	if _, ok := optOutPhrases[normalized]; ok {
		return ConsentOptedOut
	}
	if _, ok := optInPhrases[normalized]; ok {
		return ConsentOptedIn
	}
	return ConsentUnchanged
}

func normalizeConsentText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		switch r {
		case 'á':
			return 'a'
		case 'é':
			return 'e'
		case 'í':
			return 'i'
		case 'ó':
			return 'o'
		case 'ú', 'ü':
			return 'u'
		case 'ñ':
			return 'n'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func NormalizeDeliveryStatus(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "sent":
		return "SENT", true
	case "delivered":
		return "DELIVERED", true
	case "read":
		return "READ", true
	case "failed":
		return "FAILED", true
	default:
		return "", false
	}
}

type Readiness struct {
	Mode                          string   `json:"mode"`
	GraphAPIVersion               string   `json:"graph_api_version"`
	SenderIdentifiersConfigured   bool     `json:"sender_identifiers_configured"`
	WebhookVerificationConfigured bool     `json:"webhook_verification_configured"`
	SignatureValidationConfigured bool     `json:"signature_validation_configured"`
	AccessTokenConfigured         bool     `json:"access_token_configured"`
	OutboundEnabled               bool     `json:"outbound_enabled"`
	LocalDeliveryAvailable        bool     `json:"local_delivery_available"`
	CloudDeliveryAllowed          bool     `json:"cloud_delivery_allowed"`
	Missing                       []string `json:"missing"`
}

func BuildReadiness(mode, version, phoneNumberID, wabaID, accessToken, appSecret, verifyToken string, outbound bool) Readiness {
	result := Readiness{
		Mode:                          mode,
		GraphAPIVersion:               version,
		SenderIdentifiersConfigured:   strings.TrimSpace(phoneNumberID) != "" && strings.TrimSpace(wabaID) != "",
		WebhookVerificationConfigured: strings.TrimSpace(verifyToken) != "",
		SignatureValidationConfigured: strings.TrimSpace(appSecret) != "",
		AccessTokenConfigured:         strings.TrimSpace(accessToken) != "",
		OutboundEnabled:               outbound,
		LocalDeliveryAvailable:        mode == "local_mock" && outbound,
		CloudDeliveryAllowed:          false,
	}
	checks := []struct {
		ready bool
		name  string
	}{
		{result.SenderIdentifiersConfigured, "sender_identifiers"},
		{result.WebhookVerificationConfigured, "webhook_verify_token"},
		{result.SignatureValidationConfigured, "app_secret"},
		{result.AccessTokenConfigured, "access_token"},
	}
	for _, check := range checks {
		if !check.ready {
			result.Missing = append(result.Missing, check.name)
		}
	}
	return result
}
