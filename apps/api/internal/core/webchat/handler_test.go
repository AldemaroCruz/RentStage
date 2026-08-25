package webchat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerMapsInvalidTokenToNotFound(
	t *testing.T,
) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/chat",
		nil,
	)
	handler := &Handler{}

	if !handler.writeFailure(
		recorder,
		request,
		nil,
		ErrInvalidToken,
	) {
		t.Fatal("expected handled failure")
	}

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected 404, got %d",
			recorder.Code,
		)
	}
}

func TestHandlerMapsRateLimit(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/chat",
		nil,
	)
	handler := &Handler{}

	if !handler.writeFailure(
		recorder,
		request,
		nil,
		ErrRateLimited,
	) {
		t.Fatal("expected handled failure")
	}

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"expected 429, got %d",
			recorder.Code,
		)
	}

	if recorder.Header().Get("Retry-After") != "3600" {
		t.Fatal("expected Retry-After header")
	}
}

func TestPublicChatHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()

	setPublicChatHeaders(recorder)

	if recorder.Header().Get("Cache-Control") !=
		"no-store, max-age=0" {
		t.Fatal("expected no-store cache policy")
	}

	if recorder.Header().Get("Referrer-Policy") !=
		"no-referrer" {
		t.Fatal("expected no-referrer policy")
	}
}
