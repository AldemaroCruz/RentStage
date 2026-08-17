package authn

import (
	"net/http/httptest"
	"testing"

	"github.com/rentstage/rentstage/apps/api/internal/config"
)

func TestCSRFDoubleSubmit(t *testing.T) {
	cfg := config.Config{CSRFCookieName: "rentstage_csrf"}
	recorder := httptest.NewRecorder()
	token, err := IssueCSRFToken(recorder, cfg)
	if err != nil {
		t.Fatalf("IssueCSRFToken: %v", err)
	}
	request := httptest.NewRequest("POST", "/api/v1/customers", nil)
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}
	request.Header.Set("X-CSRF-Token", token)
	if !ValidCSRF(request, cfg) {
		t.Fatal("matching CSRF cookie and header should be accepted")
	}
	request.Header.Set("X-CSRF-Token", "wrong")
	if ValidCSRF(request, cfg) {
		t.Fatal("mismatched CSRF header should be rejected")
	}
}
