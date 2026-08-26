package webchat

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func salesBriefRequest() DraftRequest {
	return DraftRequest{
		Kind: DraftKindFollowUp,
		PreviousMessages: []DraftConversationMessage{
			{
				Role: DraftMessageRoleCustomer,
				Body: "Será un evento corporativo en San Salvador.",
			},
			{
				Role: DraftMessageRoleTeam,
				Body: "¿Cuál es su presupuesto aprobado?",
			},
		},
		CustomerMessage: "Será para 90 personas y todavía no tengo presupuesto.",
	}
}

func TestNormalizeDraftSalesBriefAcceptsCustomerGroundedSignals(
	t *testing.T,
) {
	brief, err := NormalizeDraftSalesBrief(
		DraftSalesBrief{
			Signals: []DraftSalesSignal{
				{
					Kind:  DraftSalesSignalEventType,
					Value: " evento corporativo ",
				},
				{
					Kind:  DraftSalesSignalLocation,
					Value: "San Salvador",
				},
				{
					Kind:  DraftSalesSignalGuestCount,
					Value: "90 personas",
				},
				{
					Kind:  DraftSalesSignalBudget,
					Value: "todavía no tengo presupuesto",
				},
			},
			MissingFields: []DraftSalesMissingField{
				DraftSalesMissingEventDate,
			},
			NextQuestion: " ¿Qué fecha tiene prevista para el evento? ",
		},
		salesBriefRequest(),
	)
	if err != nil {
		t.Fatalf("normalize sales brief: %v", err)
	}
	if len(brief.Signals) != 4 {
		t.Fatalf("unexpected signal count: %d", len(brief.Signals))
	}
	if brief.Signals[0].Value != "evento corporativo" {
		t.Fatalf("unexpected normalized signal: %#v", brief.Signals[0])
	}
	if len(brief.MissingFields) != 1 ||
		brief.MissingFields[0] != DraftSalesMissingEventDate {
		t.Fatalf("unexpected missing fields: %#v", brief.MissingFields)
	}
	if brief.NextQuestion != "¿Qué fecha tiene prevista para el evento?" {
		t.Fatalf("unexpected next question: %q", brief.NextQuestion)
	}
}

func TestNormalizeDraftSalesBriefAcceptsEmptySummary(t *testing.T) {
	brief, err := NormalizeDraftSalesBrief(
		DraftSalesBrief{},
		DraftRequest{},
	)
	if err != nil {
		t.Fatalf("normalize empty sales brief: %v", err)
	}
	if brief.Signals == nil || brief.MissingFields == nil {
		t.Fatalf("expected non-nil empty lists: %#v", brief)
	}
}

func TestNormalizeDraftSalesBriefRejectsUntrustedValues(t *testing.T) {
	tests := []struct {
		name    string
		brief   DraftSalesBrief
		request DraftRequest
	}{
		{
			name: "invented signal",
			brief: DraftSalesBrief{
				Signals: []DraftSalesSignal{
					{Kind: DraftSalesSignalLocation, Value: "Santa Ana"},
				},
			},
			request: salesBriefRequest(),
		},
		{
			name: "team message is not customer evidence",
			brief: DraftSalesBrief{
				Signals: []DraftSalesSignal{
					{
						Kind:  DraftSalesSignalBudget,
						Value: "presupuesto aprobado",
					},
				},
			},
			request: DraftRequest{
				PreviousMessages: []DraftConversationMessage{
					{
						Role: DraftMessageRoleTeam,
						Body: "¿Tiene presupuesto aprobado?",
					},
				},
			},
		},
		{
			name: "unsupported signal kind",
			brief: DraftSalesBrief{
				Signals: []DraftSalesSignal{
					{Kind: "PHONE", Value: "90 personas"},
				},
			},
			request: salesBriefRequest(),
		},
		{
			name: "duplicate signal kind",
			brief: DraftSalesBrief{
				Signals: []DraftSalesSignal{
					{Kind: DraftSalesSignalLocation, Value: "San Salvador"},
					{Kind: DraftSalesSignalLocation, Value: "San Salvador"},
				},
			},
			request: salesBriefRequest(),
		},
		{
			name: "oversized signal",
			brief: DraftSalesBrief{
				Signals: []DraftSalesSignal{
					{
						Kind: DraftSalesSignalLocation,
						Value: strings.Repeat(
							"a",
							MaximumDraftSalesSignalRunes+1,
						),
					},
				},
			},
			request: DraftRequest{
				CustomerMessage: strings.Repeat(
					"a",
					MaximumDraftSalesSignalRunes+1,
				),
			},
		},
		{
			name: "unsupported missing field",
			brief: DraftSalesBrief{
				MissingFields: []DraftSalesMissingField{"PHONE"},
				NextQuestion:  "¿Cuál es su teléfono?",
			},
		},
		{
			name: "duplicate missing field",
			brief: DraftSalesBrief{
				MissingFields: []DraftSalesMissingField{
					DraftSalesMissingEventDate,
					DraftSalesMissingEventDate,
				},
				NextQuestion: "¿Qué fecha tiene prevista?",
			},
		},
		{
			name: "missing field without question",
			brief: DraftSalesBrief{
				MissingFields: []DraftSalesMissingField{
					DraftSalesMissingEventDate,
				},
			},
		},
		{
			name: "question without missing field",
			brief: DraftSalesBrief{
				NextQuestion: "¿Qué fecha tiene prevista?",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeDraftSalesBrief(test.brief, test.request)
			if !errors.Is(err, ErrInvalidDraft) {
				t.Fatalf("expected invalid draft error, got %v", err)
			}
		})
	}
}

func TestGenerateDraftFallsBackForUngroundedSalesSignal(t *testing.T) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			result: DraftResult{
				Body:   "El equipo confirmará los detalles.",
				Engine: "TEST_PROVIDER",
				Model:  "TEST_MODEL",
				SalesBrief: DraftSalesBrief{
					Signals: []DraftSalesSignal{
						{
							Kind:  DraftSalesSignalLocation,
							Value: "Santa Ana",
						},
					},
				},
			},
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		salesBriefRequest(),
	)
	if err != nil {
		t.Fatalf("generate protected draft: %v", err)
	}
	if !result.UsedFallback ||
		result.FallbackReason != DraftFallbackReasonInvalidResponse {
		t.Fatalf("unexpected fallback: %#v", result)
	}
	if result.SalesBrief.Signals == nil ||
		len(result.SalesBrief.Signals) != 0 ||
		result.SalesBrief.MissingFields == nil ||
		len(result.SalesBrief.MissingFields) != 0 ||
		result.SalesBrief.NextQuestion != "" {
		t.Fatalf("fallback leaked sales brief: %#v", result.SalesBrief)
	}
}

func TestEncodeDraftSalesBriefUsesStableEmptyArrays(t *testing.T) {
	encoded, err := encodeDraftSalesBrief(DraftSalesBrief{})
	if err != nil {
		t.Fatalf("encode sales brief: %v", err)
	}
	if encoded !=
		`{"signals":[],"missing_fields":[],"next_question":""}` {
		t.Fatalf("unexpected encoded sales brief: %q", encoded)
	}
}
