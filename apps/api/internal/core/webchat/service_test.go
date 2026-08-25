package webchat

import (
	"strings"
	"testing"
)

const validClientMessageID = "11111111-1111-4111-8111-111111111111"

func TestNormalizeCreateSession(t *testing.T) {
	email := "  CUSTOMER@EXAMPLE.COM "

	result, fields := normalizeCreateSession(CreateSessionInput{
		ContactName:     "  Ana   Martínez  ",
		ContactEmail:    &email,
		Message:         "  Necesito sonido para una boda.  ",
		ClientMessageID: validClientMessageID,
		ConsentAccepted: true,
	}, publicConfiguration{TermsVersion: " 1.0 "})

	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if result.ContactName != "Ana Martínez" {
		t.Fatalf("unexpected contact name: %q", result.ContactName)
	}
	if result.ContactEmail == nil ||
		*result.ContactEmail != "customer@example.com" {
		t.Fatalf("unexpected contact email: %#v", result.ContactEmail)
	}
	if result.Message != "Necesito sonido para una boda." {
		t.Fatalf("unexpected message: %q", result.Message)
	}
	if result.TermsVersion != "1.0" {
		t.Fatalf("unexpected terms version: %q", result.TermsVersion)
	}
}

func TestNormalizeCreateSessionAcceptsEmptyOptionalEmail(t *testing.T) {
	email := "   "

	result, fields := normalizeCreateSession(CreateSessionInput{
		ContactName:     "Ana Martínez",
		ContactEmail:    &email,
		Message:         "Necesito información.",
		ClientMessageID: validClientMessageID,
		ConsentAccepted: true,
	}, publicConfiguration{TermsVersion: "1.0"})

	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if result.ContactEmail != nil {
		t.Fatalf("expected nil email, got %#v", result.ContactEmail)
	}
}

func TestNormalizeCreateSessionRejectsInvalidInput(t *testing.T) {
	email := "Customer <customer@example.com>"

	_, fields := normalizeCreateSession(CreateSessionInput{
		ContactName:     "A",
		ContactEmail:    &email,
		Message:         strings.Repeat("á", MaximumMessageLength+1),
		ClientMessageID: "not-a-uuid",
		ConsentAccepted: false,
		Website:         "https://spam.example",
	}, publicConfiguration{TermsVersion: "1.0"})

	for _, field := range []string{
		"contact_name",
		"contact_email",
		"message",
		"client_message_id",
		"consent_accepted",
		"website",
	} {
		if fields[field] == "" {
			t.Fatalf(
				"expected validation error for %s: %#v",
				field,
				fields,
			)
		}
	}
}

func TestNormalizeSendMessage(t *testing.T) {
	result, fields := normalizeSendMessage(SendMessageInput{
		Body:            "  ¿Tienen disponibilidad?  ",
		ClientMessageID: validClientMessageID,
	})

	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if result.Body != "¿Tienen disponibilidad?" {
		t.Fatalf("unexpected message body: %q", result.Body)
	}
}

func TestNormalizeSendMessageRejectsInvalidInput(t *testing.T) {
	_, fields := normalizeSendMessage(SendMessageInput{
		Body:            " ",
		ClientMessageID: "invalid",
	})

	if fields["body"] == "" ||
		fields["client_message_id"] == "" {
		t.Fatalf("expected body and ID errors: %#v", fields)
	}
}
