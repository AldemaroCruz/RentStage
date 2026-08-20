package meta

import "testing"

func TestClassifyConsent(t *testing.T) {
	tests := map[string]ConsentDecision{
		"STOP": ConsentOptedOut,
		"No más mensajes!!!": ConsentOptedOut,
		"DEJAR DE RECIBIR": ConsentOptedOut,
		"Continuar": ConsentOptedIn,
		"quiero una cotización": ConsentUnchanged,
	}
	for input, want := range tests {
		if got := ClassifyConsent(input); got != want {
			t.Fatalf("ClassifyConsent(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeDeliveryStatus(t *testing.T) {
	for input, want := range map[string]string{"sent": "SENT", "Delivered": "DELIVERED", "read": "READ", "failed": "FAILED"} {
		got, ok := NormalizeDeliveryStatus(input)
		if !ok || got != want {
			t.Fatalf("NormalizeDeliveryStatus(%q) = %q, %v", input, got, ok)
		}
	}
	if _, ok := NormalizeDeliveryStatus("deleted"); ok {
		t.Fatal("unknown status must be ignored")
	}
}

func TestBuildReadinessNeverEnablesCloudDelivery(t *testing.T) {
	ready := BuildReadiness("cloud", "v23.0", "phone", "waba", "token", "secret", "verify", false)
	if len(ready.Missing) != 0 || ready.CloudDeliveryAllowed || ready.OutboundEnabled {
		t.Fatalf("unexpected readiness: %+v", ready)
	}
	local := BuildReadiness("local_mock", "v-test", "phone", "waba", "token", "secret", "verify", true)
	if !local.LocalDeliveryAvailable || local.CloudDeliveryAllowed {
		t.Fatalf("unexpected local readiness: %+v", local)
	}
}
