package webchat

import (
	"strings"
	"testing"
)

func TestGenerateSessionToken(t *testing.T) {
	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if len(rawToken) != 43 {
		t.Fatalf(
			"expected 43-character token, got %d",
			len(rawToken),
		)
	}
	if len(tokenHash) != 64 {
		t.Fatalf(
			"expected 64-character hash, got %d",
			len(tokenHash),
		)
	}
	if tokenHash != hashSessionToken(rawToken) {
		t.Fatal("stored hash does not match raw token")
	}
	if strings.ContainsAny(rawToken, "+/=") {
		t.Fatalf("token is not URL-safe: %q", rawToken)
	}
}

func TestGenerateSessionTokenIsRandom(t *testing.T) {
	first, _, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generate first token: %v", err)
	}

	second, _, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generate second token: %v", err)
	}

	if first == second {
		t.Fatal("expected independently generated tokens")
	}
}

func TestHashSessionTokenTrimsTransportWhitespace(t *testing.T) {
	if hashSessionToken(" token ") !=
		hashSessionToken("token") {
		t.Fatal(
			"transport whitespace should not change the token hash",
		)
	}
}
