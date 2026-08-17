package reservation

import "testing"

func TestNormalizeManualReservationCalculatesTotalsAndSchedule(t *testing.T) {
	result, fields := normalizeCreateInput(CreateInput{
		CustomerID:     "40000000-0000-0000-0000-000000000001",
		BlockStartAt:   "2026-08-22T14:00:00-06:00",
		BlockEndAt:     "2026-08-23T02:00:00-06:00",
		EventStartAt:   "2026-08-22T18:00:00-06:00",
		EventEndAt:     "2026-08-22T23:00:00-06:00",
		DiscountAmount: 10,
		ExtraCharges:   25,
		Items: []CreateItemInput{
			{ResourceID: "20000000-0000-0000-0000-000000000001", Quantity: 2, UnitPrice: 40},
			{ResourceID: "20000000-0000-0000-0000-000000000004", Quantity: 4, UnitPrice: 8, DiscountAmount: 2},
		},
	})
	if len(fields) != 0 {
		t.Fatalf("expected valid manual reservation, got %#v", fields)
	}
	if result.Subtotal != 110 {
		t.Fatalf("subtotal = %v, want 110", result.Subtotal)
	}
	if result.Total != 125 {
		t.Fatalf("total = %v, want 125", result.Total)
	}
	if result.Items[1].LineTotal != 30 {
		t.Fatalf("second line = %v, want 30", result.Items[1].LineTotal)
	}
}

func TestNormalizeManualReservationRejectsEventOutsideBlock(t *testing.T) {
	_, fields := normalizeCreateInput(CreateInput{
		CustomerID:   "40000000-0000-0000-0000-000000000001",
		BlockStartAt: "2026-08-22T14:00:00-06:00",
		BlockEndAt:   "2026-08-22T20:00:00-06:00",
		EventStartAt: "2026-08-22T13:00:00-06:00",
		EventEndAt:   "2026-08-22T21:00:00-06:00",
		Items: []CreateItemInput{{
			ResourceID: "20000000-0000-0000-0000-000000000001",
			Quantity:   1,
			UnitPrice:  40,
		}},
	})
	if fields["event_start_at"] == "" {
		t.Fatal("expected event_start_at validation error")
	}
	if fields["event_end_at"] == "" {
		t.Fatal("expected event_end_at validation error")
	}
}

func TestNormalizeReschedule(t *testing.T) {
	result, fields := normalizeRescheduleInput(RescheduleInput{
		BlockStartAt: "2026-08-24T14:00:00-06:00",
		BlockEndAt:   "2026-08-25T02:00:00-06:00",
		EventStartAt: "2026-08-24T18:00:00-06:00",
		EventEndAt:   "2026-08-24T23:00:00-06:00",
		Reason:       "  Client changed the date.  ",
	})
	if len(fields) != 0 {
		t.Fatalf("expected valid schedule, got %#v", fields)
	}
	if result.Reason != "Client changed the date." {
		t.Fatalf("unexpected reason %q", result.Reason)
	}
	if !result.BlockEndAt.After(result.BlockStartAt) {
		t.Fatal("expected block end after start")
	}
}
