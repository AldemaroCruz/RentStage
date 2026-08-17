package reservation

import (
	"testing"
	"time"
)

func TestNormalizeReturnInput(t *testing.T) {
	input := ReturnInput{
		Notes: "  general notes  ",
		Assets: []ReturnAssetInput{
			{
				AssetID:   "30000000-0000-0000-0000-000000000001",
				Condition: " maintenance_required ",
				Notes:     "  connector noise  ",
			},
		},
	}

	result, fields := normalizeReturnInput(input)
	if len(fields) != 0 {
		t.Fatalf("expected no validation errors, got %#v", fields)
	}
	if result.Notes != "general notes" {
		t.Fatalf("unexpected normalized notes: %q", result.Notes)
	}
	if result.Assets[0].Condition != "MAINTENANCE_REQUIRED" {
		t.Fatalf("unexpected condition: %q", result.Assets[0].Condition)
	}
	if result.Assets[0].Notes != "connector noise" {
		t.Fatalf("unexpected asset notes: %q", result.Assets[0].Notes)
	}
}

func TestNormalizeReturnInputRejectsDuplicatesAndConditions(t *testing.T) {
	assetID := "30000000-0000-0000-0000-000000000001"
	_, fields := normalizeReturnInput(ReturnInput{Assets: []ReturnAssetInput{
		{AssetID: assetID, Condition: "GOOD"},
		{AssetID: assetID, Condition: "BROKEN"},
	}})
	if fields["assets.1.asset_id"] == "" {
		t.Fatal("expected duplicate asset validation error")
	}
	if fields["assets.1.condition"] == "" {
		t.Fatal("expected condition validation error")
	}
}

func TestPhysicalStatusForReturn(t *testing.T) {
	cases := map[string]string{
		"GOOD":                 "AVAILABLE",
		"MAINTENANCE_REQUIRED": "MAINTENANCE",
		"DAMAGED":              "DAMAGED",
		"LOST":                 "LOST",
	}
	for condition, expected := range cases {
		if actual := physicalStatusForReturn(condition); actual != expected {
			t.Fatalf("condition %s: expected %s, got %s", condition, expected, actual)
		}
	}
}

func TestAssignmentStatePrecedence(t *testing.T) {
	checkedOut := nowForTest()
	returned := checkedOut
	released := checkedOut

	cases := []struct {
		name     string
		item     AssignedAsset
		expected string
	}{
		{name: "assigned", item: AssignedAsset{}, expected: "ASSIGNED"},
		{name: "checked out", item: AssignedAsset{CheckedOutAt: &checkedOut}, expected: "CHECKED_OUT"},
		{name: "returned", item: AssignedAsset{CheckedOutAt: &checkedOut, ReturnedAt: &returned}, expected: "RETURNED"},
		{name: "released", item: AssignedAsset{CheckedOutAt: &checkedOut, ReturnedAt: &returned, ReleasedAt: &released}, expected: "RELEASED"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if actual := assignmentState(test.item); actual != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, actual)
			}
		})
	}
}

func nowForTest() time.Time {
	return time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
}
