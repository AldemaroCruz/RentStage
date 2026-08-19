package commercialmetrics

import "testing"

func TestPercentage(t *testing.T) {
	tests := []struct {
		name        string
		numerator   int
		denominator int
		want        float64
	}{
		{name: "whole", numerator: 3, denominator: 4, want: 75},
		{name: "rounded", numerator: 2, denominator: 3, want: 66.7},
		{name: "empty", numerator: 1, denominator: 0, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := percentage(test.numerator, test.denominator); got != test.want {
				t.Fatalf("percentage(%d, %d) = %v, want %v", test.numerator, test.denominator, got, test.want)
			}
		})
	}
}

func TestFinalizeBuildsTransparentCommercialStages(t *testing.T) {
	report := Report{
		Overview: Overview{
			PublicRequests:           4,
			AssistantConversations:   6,
			QuotesCreated:            8,
			QuotesAccepted:           3,
			QuotesRejected:           1,
			ReservationsCreated:      2,
			QuoteReservationsCreated: 2,
			InvoicesIssued:           1,
		},
		Outcomes: ReservationOutcomes{Completed: 4, Cancelled: 1},
	}
	report.finalize()

	if report.Overview.Inquiries != 10 {
		t.Fatalf("inquiries = %d, want 10", report.Overview.Inquiries)
	}
	if report.Overview.QuoteAcceptanceRate != 75 {
		t.Fatalf("quote acceptance = %v, want 75", report.Overview.QuoteAcceptanceRate)
	}
	if report.Overview.QuoteToReservationRate != 66.7 {
		t.Fatalf("quote to reservation = %v, want 66.7", report.Overview.QuoteToReservationRate)
	}
	if report.Outcomes.CancellationRate != 20 {
		t.Fatalf("cancellation rate = %v, want 20", report.Outcomes.CancellationRate)
	}
	if len(report.Funnel) != 5 || report.Funnel[0].Count != 10 || report.Funnel[4].Count != 1 {
		t.Fatalf("unexpected funnel: %+v", report.Funnel)
	}
}

func TestReportDays(t *testing.T) {
	for _, value := range []string{"", "7", "30", "90"} {
		if _, valid := reportDays(value); !valid {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"0", "14", "thirty"} {
		if _, valid := reportDays(value); valid {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}
