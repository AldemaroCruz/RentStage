package webchat

import "testing"

func TestNormalizeSessionAccess(t *testing.T) {
	rawToken, expectedHash, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	actualHash, err := normalizeSessionAccess(
		validClientMessageID,
		rawToken,
	)
	if err != nil {
		t.Fatalf("normalize session access: %v", err)
	}
	if actualHash != expectedHash {
		t.Fatalf("unexpected token hash: %q", actualHash)
	}
}

func TestNormalizeSessionAccessRejectsInvalidValues(
	t *testing.T,
) {
	_, err := normalizeSessionAccess("invalid", "short")
	if err != ErrInvalidToken {
		t.Fatalf(
			"expected invalid token error, got %v",
			err,
		)
	}
}

func TestResponseDrafts(t *testing.T) {
	if draft := initialResponseDraft("Ana"); draft == "" {
		t.Fatal("expected initial response draft")
	}
	if draft := followUpResponseDraft(); draft == "" {
		t.Fatal("expected follow-up response draft")
	}
}
