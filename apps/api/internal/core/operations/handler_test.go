package operations

import "testing"

func TestParseStatuses(t *testing.T) {
	statuses, message := parseStatuses("confirmed, preparing,CONFIRMED")
	if message != "" {
		t.Fatalf("unexpected validation message: %s", message)
	}
	if len(statuses) != 2 || statuses[0] != "CONFIRMED" || statuses[1] != "PREPARING" {
		t.Fatalf("unexpected statuses: %#v", statuses)
	}
}

func TestParseStatusesRejectsUnsupportedValue(t *testing.T) {
	if _, message := parseStatuses("CONFIRMED,UNKNOWN"); message == "" {
		t.Fatal("expected unsupported status validation message")
	}
}
