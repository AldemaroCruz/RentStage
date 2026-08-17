package availability

import "testing"

func TestNormalizeAvailability(t *testing.T) {
	input := CheckInput{
		StartAt: "2026-08-20T14:00:00-06:00",
		EndAt:   "2026-08-21T02:00:00-06:00",
		Items: []ItemInput{
			{ResourceID: "20000000-0000-0000-0000-000000000001", Quantity: 2},
		},
	}
	normalized, fields := Normalize(input)
	if len(fields) != 0 {
		t.Fatalf("expected valid input, got fields: %#v", fields)
	}
	if len(normalized.Items) != 1 || normalized.Items[0].Quantity != 2 {
		t.Fatalf("unexpected normalized items: %#v", normalized.Items)
	}
	if !normalized.EndAt.After(normalized.StartAt) {
		t.Fatal("expected end after start")
	}
}

func TestNormalizeAvailabilityRejectsDuplicatesAndInvalidPeriod(t *testing.T) {
	resourceID := "20000000-0000-0000-0000-000000000001"
	_, fields := Normalize(CheckInput{
		StartAt: "2026-08-21T02:00:00-06:00",
		EndAt:   "2026-08-20T14:00:00-06:00",
		Items: []ItemInput{
			{ResourceID: resourceID, Quantity: 1},
			{ResourceID: resourceID, Quantity: 1},
		},
	})
	if fields["end_at"] == "" {
		t.Fatal("expected invalid period field")
	}
	if fields["items[1].resource_id"] == "" {
		t.Fatal("expected duplicate resource field")
	}
}
