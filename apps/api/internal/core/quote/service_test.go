package quote

import "testing"

func TestNormalizeQuoteCalculatesHistoricalTotals(t *testing.T) {
	result, fields := normalize(CreateInput{
		CustomerID:     "40000000-0000-0000-0000-000000000001",
		StartAt:        "2026-08-22T20:00:00Z",
		EndAt:          "2026-08-23T08:00:00Z",
		DiscountAmount: 10,
		ExtraCharges:   30,
		Items: []ItemInput{
			{
				ResourceID:     "20000000-0000-0000-0000-000000000001",
				Quantity:       2,
				UnitPrice:      40,
				DiscountAmount: 5,
			},
			{
				ResourceID: "20000000-0000-0000-0000-000000000002",
				Quantity:   1,
				UnitPrice:  65,
			},
		},
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if result.Subtotal != 140 {
		t.Fatalf("subtotal = %v, want 140", result.Subtotal)
	}
	if result.Total != 160 {
		t.Fatalf("total = %v, want 160", result.Total)
	}
	if result.Items[0].LineTotal != 75 {
		t.Fatalf("first line total = %v, want 75", result.Items[0].LineTotal)
	}
}

func TestNormalizeQuoteRejectsOverlappingResourceLines(t *testing.T) {
	resourceID := "20000000-0000-0000-0000-000000000001"
	_, fields := normalize(CreateInput{
		CustomerID: "40000000-0000-0000-0000-000000000001",
		StartAt:    "2026-08-22T20:00:00Z",
		EndAt:      "2026-08-23T08:00:00Z",
		Items: []ItemInput{
			{ResourceID: resourceID, Quantity: 1, UnitPrice: 40},
			{ResourceID: resourceID, Quantity: 1, UnitPrice: 40},
		},
	})
	if fields["items[1].resource_id"] == "" {
		t.Fatal("expected duplicate resource validation error")
	}
}
