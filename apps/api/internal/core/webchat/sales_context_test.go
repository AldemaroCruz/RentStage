package webchat

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func validDraftSalesContext() DraftSalesContext {
	capacity := 100
	packagePrice := 299.0
	resourcePrice := 40.0

	return DraftSalesContext{
		Currency:             " usd ",
		ShowPrices:           true,
		ShowResources:        true,
		QuoteRequestsEnabled: true,
		Packages: []DraftSalesPackage{
			{
				Name:          " Paquete Fiesta ",
				Description:   " Audio para 100 personas. ",
				GuestCapacity: &capacity,
				Price:         &packagePrice,
			},
		},
		Resources: []DraftSalesResource{
			{
				Name:        " JBL PRX815W ",
				Description: " Bocina activa. ",
				Category:    " Speakers ",
				Type:        " EQUIPMENT ",
				PricingUnit: " DAY ",
				Price:       &resourcePrice,
			},
		},
	}
}

func TestNormalizeDraftSalesContext(t *testing.T) {
	context, err := NormalizeDraftSalesContext(
		validDraftSalesContext(),
	)
	if err != nil {
		t.Fatalf("normalize sales context: %v", err)
	}

	if context.Currency != "USD" {
		t.Fatalf("unexpected currency: %q", context.Currency)
	}
	if context.Packages[0].Name != "Paquete Fiesta" {
		t.Fatalf(
			"unexpected package name: %q",
			context.Packages[0].Name,
		)
	}
	if context.Resources[0].Category != "Speakers" {
		t.Fatalf(
			"unexpected resource category: %q",
			context.Resources[0].Category,
		)
	}
}

func TestNormalizeDraftSalesContextAcceptsEmptyCatalog(
	t *testing.T,
) {
	context, err := NormalizeDraftSalesContext(
		DraftSalesContext{Currency: "USD"},
	)
	if err != nil {
		t.Fatalf("normalize empty sales context: %v", err)
	}
	if context.Packages == nil || context.Resources == nil {
		t.Fatal("expected empty, non-nil catalog collections")
	}
}

func TestNormalizeDraftSalesContextRejectsPrivatePrices(
	t *testing.T,
) {
	context := validDraftSalesContext()
	context.ShowPrices = false

	_, err := NormalizeDraftSalesContext(context)
	if !errors.Is(err, ErrInvalidDraft) {
		t.Fatalf("expected invalid draft error, got %v", err)
	}
}

func TestNormalizeDraftSalesContextRejectsHiddenResources(
	t *testing.T,
) {
	context := validDraftSalesContext()
	context.ShowResources = false

	_, err := NormalizeDraftSalesContext(context)
	if !errors.Is(err, ErrInvalidDraft) {
		t.Fatalf("expected invalid draft error, got %v", err)
	}
}

func TestNormalizeDraftSalesContextRejectsInvalidValues(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(*DraftSalesContext)
	}{
		{
			name: "currency",
			mutate: func(context *DraftSalesContext) {
				context.Currency = "US1"
			},
		},
		{
			name: "description",
			mutate: func(context *DraftSalesContext) {
				context.Packages[0].Description = strings.Repeat(
					"á",
					MaximumDraftSalesDescriptionRunes+1,
				)
			},
		},
		{
			name: "capacity",
			mutate: func(context *DraftSalesContext) {
				invalid := 0
				context.Packages[0].GuestCapacity = &invalid
			},
		},
		{
			name: "negative price",
			mutate: func(context *DraftSalesContext) {
				invalid := -1.0
				context.Resources[0].Price = &invalid
			},
		},
		{
			name: "non finite price",
			mutate: func(context *DraftSalesContext) {
				invalid := math.Inf(1)
				context.Resources[0].Price = &invalid
			},
		},
		{
			name: "total rune budget",
			mutate: func(context *DraftSalesContext) {
				context.Packages[0].Name = strings.Repeat(
					"x",
					MaximumDraftSalesContextRunes,
				)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context := validDraftSalesContext()
			test.mutate(&context)

			_, err := NormalizeDraftSalesContext(context)
			if !errors.Is(err, ErrInvalidDraft) {
				t.Fatalf(
					"expected invalid draft error, got %v",
					err,
				)
			}
		})
	}
}

func TestNormalizeDraftSalesContextRejectsExcessItems(
	t *testing.T,
) {
	context := validDraftSalesContext()
	context.Packages = make(
		[]DraftSalesPackage,
		MaximumDraftSalesPackages+1,
	)

	_, err := NormalizeDraftSalesContext(context)
	if !errors.Is(err, ErrInvalidDraft) {
		t.Fatalf("expected invalid draft error, got %v", err)
	}
}
