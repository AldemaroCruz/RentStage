package webchat

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func groundingSalesContext() DraftSalesContext {
	packagePrice := 299.0
	resourcePrice := 40.0

	return DraftSalesContext{
		Currency:      "USD",
		ShowPrices:    true,
		ShowResources: true,
		Packages: []DraftSalesPackage{
			{
				Name:        "Paquete Fiesta 100 personas",
				Description: "Sistema de audio para eventos.",
				Price:       &packagePrice,
			},
		},
		Resources: []DraftSalesResource{
			{
				Name:        "JBL PRX815W",
				Description: "Bocina activa para sonido principal.",
				Category:    "Speakers",
				Type:        "EQUIPMENT",
				PricingUnit: "DAY",
				Price:       &resourcePrice,
			},
		},
	}
}

func TestNormalizeDraftGroundingReferencesUsesCanonicalCatalogNames(
	t *testing.T,
) {
	references, err := NormalizeDraftGroundingReferences(
		[]DraftGroundingReference{
			{
				Kind: DraftGroundingKindPackage,
				Name: "  paquete fiesta 100 PERSONAS  ",
			},
			{
				Kind: DraftGroundingKindResource,
				Name: "jbl prx815w",
			},
		},
		groundingSalesContext(),
	)
	if err != nil {
		t.Fatalf("normalize grounding references: %v", err)
	}

	if len(references) != 2 {
		t.Fatalf("unexpected reference count: %d", len(references))
	}
	if references[0].Name != "Paquete Fiesta 100 personas" {
		t.Fatalf("unexpected package name: %q", references[0].Name)
	}
	if references[1].Name != "JBL PRX815W" {
		t.Fatalf("unexpected resource name: %q", references[1].Name)
	}
}

func TestNormalizeDraftGroundingReferencesAcceptsEmptyList(
	t *testing.T,
) {
	references, err := NormalizeDraftGroundingReferences(nil, DraftSalesContext{})
	if err != nil {
		t.Fatalf("normalize empty grounding references: %v", err)
	}
	if references == nil || len(references) != 0 {
		t.Fatalf("expected an empty non-nil list, got %#v", references)
	}
}

func TestNormalizeDraftGroundingReferencesRejectsUntrustedValues(
	t *testing.T,
) {
	tests := []struct {
		name       string
		references []DraftGroundingReference
		context    DraftSalesContext
	}{
		{
			name: "hallucinated package",
			references: []DraftGroundingReference{
				{Kind: DraftGroundingKindPackage, Name: "Paquete inexistente"},
			},
			context: groundingSalesContext(),
		},
		{
			name: "hidden resource",
			references: []DraftGroundingReference{
				{Kind: DraftGroundingKindResource, Name: "JBL PRX815W"},
			},
			context: DraftSalesContext{
				Currency:      "USD",
				ShowResources: false,
			},
		},
		{
			name: "unsupported kind",
			references: []DraftGroundingReference{
				{Kind: "INTERNAL_RESOURCE", Name: "JBL PRX815W"},
			},
			context: groundingSalesContext(),
		},
		{
			name: "duplicate reference",
			references: []DraftGroundingReference{
				{Kind: DraftGroundingKindPackage, Name: "Paquete Fiesta 100 personas"},
				{Kind: DraftGroundingKindPackage, Name: "paquete fiesta 100 personas"},
			},
			context: groundingSalesContext(),
		},
		{
			name: "too many references",
			references: func() []DraftGroundingReference {
				items := make(
					[]DraftGroundingReference,
					MaximumDraftGroundingReferences+1,
				)
				for index := range items {
					items[index] = DraftGroundingReference{
						Kind: DraftGroundingKindPackage,
						Name: "Paquete Fiesta 100 personas",
					}
				}
				return items
			}(),
			context: groundingSalesContext(),
		},
		{
			name: "oversized name",
			references: []DraftGroundingReference{
				{
					Kind: DraftGroundingKindPackage,
					Name: strings.Repeat("a", MaximumDraftSalesNameRunes+1),
				},
			},
			context: groundingSalesContext(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeDraftGroundingReferences(
				test.references,
				test.context,
			)
			if !errors.Is(err, ErrInvalidDraft) {
				t.Fatalf("expected invalid draft error, got %v", err)
			}
		})
	}
}

func TestGenerateDraftFallsBackWhenProviderInventsReference(
	t *testing.T,
) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			result: DraftResult{
				Body:   "El paquete inexistente podría servir.",
				Engine: "TEST_PROVIDER",
				Model:  "TEST_MODEL",
				GroundingReferences: []DraftGroundingReference{
					{
						Kind: DraftGroundingKindPackage,
						Name: "Paquete inexistente",
					},
				},
			},
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{
			Kind:         DraftKindFollowUp,
			SalesContext: groundingSalesContext(),
		},
	)
	if err != nil {
		t.Fatalf("generate protected draft: %v", err)
	}
	if !result.UsedFallback ||
		result.FallbackReason != DraftFallbackReasonInvalidResponse {
		t.Fatalf("unexpected fallback result: %#v", result)
	}
	if result.GroundingReferences == nil ||
		len(result.GroundingReferences) != 0 {
		t.Fatalf("fallback must not retain references: %#v", result)
	}
}

func TestGenerateDraftReturnsCanonicalValidatedReferences(
	t *testing.T,
) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			result: DraftResult{
				Body:   "El paquete publicado puede servir como referencia.",
				Engine: "TEST_PROVIDER",
				Model:  "TEST_MODEL",
				GroundingReferences: []DraftGroundingReference{
					{
						Kind: DraftGroundingKindPackage,
						Name: "paquete fiesta 100 PERSONAS",
					},
				},
			},
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{
			Kind:         DraftKindFollowUp,
			SalesContext: groundingSalesContext(),
		},
	)
	if err != nil {
		t.Fatalf("generate grounded draft: %v", err)
	}
	if result.UsedFallback {
		t.Fatalf("did not expect fallback: %#v", result)
	}
	if len(result.GroundingReferences) != 1 ||
		result.GroundingReferences[0].Name !=
			"Paquete Fiesta 100 personas" {
		t.Fatalf("unexpected validated references: %#v", result)
	}
}

func TestEncodeDraftGroundingReferencesUsesJSONArray(t *testing.T) {
	encoded, err := encodeDraftGroundingReferences(nil)
	if err != nil {
		t.Fatalf("encode grounding references: %v", err)
	}
	if encoded != "[]" {
		t.Fatalf("unexpected encoded references: %q", encoded)
	}
}
