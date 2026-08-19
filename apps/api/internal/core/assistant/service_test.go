package assistant

import (
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
