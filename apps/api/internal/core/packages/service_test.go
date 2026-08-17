package packages

import "testing"

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Paquete fiesta 100 personas":  "paquete-fiesta-100-personas",
		"  Audio & Iluminación / VIP ": "audio-iluminacion-vip",
		"Niñez y música":               "ninez-y-musica",
	}
	for input, want := range tests {
		if got := slugify(input); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeRejectsDuplicateResources(t *testing.T) {
	active := true
	input := CreateInput{
		Name:        "Fiesta",
		PricingMode: PricingModeSumItems,
		Active:      &active,
		Items: []ItemInput{
			{ResourceID: "20000000-0000-0000-0000-000000000001", Quantity: 2},
			{ResourceID: "20000000-0000-0000-0000-000000000001", Quantity: 1},
		},
	}
	_, fields := normalize(input)
	if fields["items[1].resource_id"] == "" {
		t.Fatal("expected duplicate resource validation")
	}
}

func TestFixedDiscountTemplatePreservesTotal(t *testing.T) {
	fixed := 299.00
	detail := Detail{
		Summary: Summary{
			ID:              "70000000-0000-0000-0000-000000000001",
			Name:            "Paquete fiesta 100 personas",
			PricingMode:     PricingModeFixed,
			FixedPrice:      &fixed,
			CalculatedPrice: 331,
			EffectivePrice:  fixed,
			Active:          true,
			Ready:           true,
		},
		Items: []Item{
			{ResourceID: "1", ResourceName: "Speakers", Description: "Speakers", Quantity: 2, UnitPrice: 40, ResourceActive: true},
			{ResourceID: "2", ResourceName: "Subs", Description: "Subs", Quantity: 2, UnitPrice: 65, ResourceActive: true},
			{ResourceID: "3", ResourceName: "Mixer", Description: "Mixer", Quantity: 1, UnitPrice: 85, ResourceActive: true},
			{ResourceID: "4", ResourceName: "Mics", Description: "Mics", Quantity: 2, UnitPrice: 8, ResourceActive: true},
			{ResourceID: "5", ResourceName: "Cableado", Description: "Cableado", Quantity: 1, UnitPrice: 20, ResourceActive: true},
		},
	}
	template := buildQuoteTemplate(detail, 3)
	if template.EffectivePrice != 897 {
		t.Fatalf("template effective price = %.2f, want 897.00", template.EffectivePrice)
	}
	var total float64
	for _, item := range template.Items {
		gross := roundMoney(float64(item.Quantity) * item.UnitPrice)
		if roundMoney(gross-item.DiscountAmount) != item.LineTotal {
			t.Fatalf("line consistency failed for %+v", item)
		}
		total = roundMoney(total + item.LineTotal)
	}
	if total != template.EffectivePrice {
		t.Fatalf("line total sum = %.2f, want %.2f", total, template.EffectivePrice)
	}
	if template.DiscountAmount != 96 {
		t.Fatalf("discount = %.2f, want 96.00", template.DiscountAmount)
	}
}

func TestFixedPremiumUsesExtraCharges(t *testing.T) {
	fixed := 120.00
	detail := Detail{
		Summary: Summary{ID: "p", Name: "P", PricingMode: PricingModeFixed, CalculatedPrice: 100, EffectivePrice: fixed, FixedPrice: &fixed},
		Items:   []Item{{ResourceID: "r", ResourceName: "Service", Description: "Service", Quantity: 1, UnitPrice: 100}},
	}
	template := buildQuoteTemplate(detail, 2)
	if template.CalculatedPrice != 200 || template.EffectivePrice != 240 || template.ExtraCharges != 40 {
		t.Fatalf("unexpected premium template: %+v", template)
	}
}

func TestSumItemsTemplateUsesPackageQuantity(t *testing.T) {
	detail := Detail{
		Summary: Summary{ID: "p", Name: "P", PricingMode: PricingModeSumItems, CalculatedPrice: 24, EffectivePrice: 24},
		Items:   []Item{{ResourceID: "r", ResourceName: "Mic", Description: "Mic", Quantity: 2, UnitPrice: 12}},
	}
	template := buildQuoteTemplate(detail, 2)
	if len(template.Items) != 1 || template.Items[0].Quantity != 4 || template.Items[0].LineTotal != 48 {
		t.Fatalf("unexpected template: %+v", template)
	}
}

func TestNormalizePricingRules(t *testing.T) {
	fixed := 125.559
	active := false

	normalized, fields := normalize(CreateInput{
		Name:        "Paquete corporativo",
		PricingMode: PricingModeSumItems,
		FixedPrice:  &fixed,
		Active:      &active,
		Items: []ItemInput{{
			ResourceID: "20000000-0000-0000-0000-000000000001",
			Quantity:   1,
		}},
	})
	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %+v", fields)
	}
	if normalized.FixedPrice != nil {
		t.Fatal("SUM_ITEMS must clear fixed_price")
	}
	if normalized.Active {
		t.Fatal("explicit active=false was not preserved")
	}

	_, fields = normalize(CreateInput{
		Name:        "Paquete fijo",
		PricingMode: PricingModeFixed,
		Items: []ItemInput{{
			ResourceID: "20000000-0000-0000-0000-000000000001",
			Quantity:   1,
		}},
	})
	if fields["fixed_price"] == "" {
		t.Fatal("FIXED must require fixed_price")
	}
}

func TestQuoteTemplateMoneyInvariant(t *testing.T) {
	items := []Item{
		{ResourceID: "1", ResourceName: "A", Description: "A", Quantity: 3, UnitPrice: 17.31, ResourceActive: true},
		{ResourceID: "2", ResourceName: "B", Description: "B", Quantity: 2, UnitPrice: 41.27, ResourceActive: true},
		{ResourceID: "3", ResourceName: "C", Description: "C", Quantity: 1, UnitPrice: 8.09, ResourceActive: true},
	}
	calculated := 0.0
	for _, item := range items {
		calculated = roundMoney(calculated + float64(item.Quantity)*item.UnitPrice)
	}

	fixedPrices := []float64{0, 0.01, 73.19, calculated - 0.01, calculated, calculated + 19.95}
	for _, fixed := range fixedPrices {
		for quantity := 1; quantity <= 7; quantity++ {
			detail := Detail{
				Summary: Summary{
					ID:              "70000000-0000-0000-0000-000000000001",
					Name:            "Invariant",
					PricingMode:     PricingModeFixed,
					FixedPrice:      &fixed,
					CalculatedPrice: calculated,
					EffectivePrice:  fixed,
					Active:          true,
					Ready:           true,
				},
				Items: items,
			}
			template := buildQuoteTemplate(detail, quantity)
			lineTotal := 0.0
			for _, item := range template.Items {
				gross := roundMoney(float64(item.Quantity) * item.UnitPrice)
				if item.DiscountAmount < 0 || item.DiscountAmount > gross {
					t.Fatalf("invalid line discount for fixed %.2f x%d: %+v", fixed, quantity, item)
				}
				if roundMoney(gross-item.DiscountAmount) != item.LineTotal {
					t.Fatalf("line total mismatch for fixed %.2f x%d: %+v", fixed, quantity, item)
				}
				lineTotal = roundMoney(lineTotal + item.LineTotal)
			}
			if got := roundMoney(lineTotal + template.ExtraCharges); got != template.EffectivePrice {
				t.Fatalf("commercial total invariant failed for fixed %.2f x%d: got %.2f, want %.2f (%+v)", fixed, quantity, got, template.EffectivePrice, template)
			}
		}
	}
}

func TestValidateQuantityBounds(t *testing.T) {
	for _, quantity := range []int{0, -1, 101} {
		if validateQuantity(quantity)["quantity"] == "" {
			t.Fatalf("quantity %d should be rejected", quantity)
		}
	}
	for _, quantity := range []int{1, 100} {
		if fields := validateQuantity(quantity); len(fields) != 0 {
			t.Fatalf("quantity %d should be accepted: %+v", quantity, fields)
		}
	}
}

func TestPackageQuantityLimit(t *testing.T) {
	tests := []struct {
		name  string
		items []Item
		want  int
	}{
		{name: "empty", items: nil, want: 100},
		{name: "ordinary package", items: []Item{{Quantity: 2}, {Quantity: 1}}, want: 100},
		{name: "large component", items: []Item{{Quantity: 250}}, want: 40},
		{name: "single maximum component", items: []Item{{Quantity: 10_000}}, want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := packageQuantityLimit(test.items); got != test.want {
				t.Fatalf("packageQuantityLimit() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestValidatePackageQuantityUsesComponentLimit(t *testing.T) {
	detail := Detail{Items: []Item{
		{Quantity: 250},
		{Quantity: 80},
	}}
	if fields := validatePackageQuantity(detail, 40); len(fields) != 0 {
		t.Fatalf("40 package units should be accepted: %+v", fields)
	}
	if fields := validatePackageQuantity(detail, 41); fields["quantity"] == "" {
		t.Fatal("41 package units should exceed the 10,000-unit component bound")
	}
}
