package assistant

import (
	"strings"
	"testing"

	"github.com/rentstage/rentstage/apps/api/internal/core/packages"
)

func TestRankPackagesPrefersReadyCapacityAndIntent(t *testing.T) {
	hundred := 100
	fifty := 50
	input := normalizedSimulation{
		Message:    "Necesito sonido para una boda de cien personas.",
		EventType:  "Boda",
		GuestCount: 100,
	}
	items := []packages.Summary{
		{ID: "small", Name: "Audio básico", GuestCapacity: &fifty, Active: true, Ready: true},
		{ID: "archived", Name: "Boda premium", GuestCapacity: &hundred, Active: false, Ready: true},
		{ID: "match", Name: "Paquete boda y fiesta", GuestCapacity: &hundred, Active: true, Ready: true},
		{ID: "broken", Name: "Boda incompleta", GuestCapacity: &hundred, Active: true, Ready: false},
	}

	result := rankPackages(items, input)
	if len(result) != 2 {
		t.Fatalf("expected 2 usable packages, got %d", len(result))
	}
	if result[0].ID != "match" {
		t.Fatalf("expected intent and capacity match first, got %s", result[0].ID)
	}
}

func TestNormalizeSimulationRejectsIncompleteIntent(t *testing.T) {
	_, fields := normalizeSimulation(SimulateInput{
		ContactName:   "",
		ContactPhone:  "7123",
		Message:       "hola",
		EventType:     "",
		StartAt:       "tomorrow",
		EndAt:         "later",
		EventLocation: "",
		GuestCount:    0,
	})
	for _, key := range []string{"contact_name", "contact_phone", "message", "event_type", "start_at", "end_at", "event_location", "guest_count"} {
		if fields[key] == "" {
			t.Fatalf("expected validation field %s", key)
		}
	}
}

func TestFormatUSDUsesPresentationSafePrecision(t *testing.T) {
	if got := formatUSD(299); got != "$299.00" {
		t.Fatalf("unexpected money: %s", got)
	}
}

func TestDraftDemoReplyKeepsHumanControlledQuoteBoundary(t *testing.T) {
	quoteNumber := int64(42)
	detail := ConversationDetail{
		ConversationSummary: ConversationSummary{ContactName: "Ana Martínez"},
		Proposal:            &Proposal{QuoteNumber: &quoteNumber},
	}

	result := draftDemoReply(detail, "¿Puedo pagar un anticipo?")
	for _, expected := range []string{"Ana Martínez", "COT-000042", "nada se cobrará ni reservará"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %q in demo reply: %s", expected, result)
		}
	}
}

func TestDraftDemoReplyUsesSafeFallback(t *testing.T) {
	detail := ConversationDetail{
		ConversationSummary: ConversationSummary{ContactName: "Cliente Demo"},
	}
	result := draftDemoReply(detail, "Tengo otra consulta")
	if !strings.Contains(result, "Una persona del equipo revisará") {
		t.Fatalf("unexpected fallback reply: %s", result)
	}
}
