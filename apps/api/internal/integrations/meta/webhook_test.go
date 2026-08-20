package meta

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type recordingProcessor struct {
	events WebhookEvents
}

func (p *recordingProcessor) ProcessMetaWebhook(_ context.Context, events WebhookEvents) (ProcessResult, error) {
	p.events = events
	return ProcessResult{InboundProcessed: len(events.Inbound), StatusesProcessed: len(events.Statuses)}, nil
}

func TestValidateSignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account","entry":[]}`)
	signature := SignForTest(body, "local-secret")
	if err := ValidateSignature(body, signature, "local-secret"); err != nil {
		t.Fatalf("ValidateSignature returned an error: %v", err)
	}
	if err := ValidateSignature([]byte("modified"), signature, "local-secret"); err == nil {
		t.Fatal("expected a modified body to be rejected")
	}
}

func TestWebhookVerificationReturnsRawChallenge(t *testing.T) {
	handler := NewWebhookHandler("verify-me", "secret", &recordingProcessor{})
	request := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=12345", nil)
	recorder := httptest.NewRecorder()
	handler.Verify(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "12345" {
		t.Fatalf("verification response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestWebhookVerificationRejectsWrongToken(t *testing.T) {
	handler := NewWebhookHandler("verify-me", "secret", &recordingProcessor{})
	request := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=12345", nil)
	recorder := httptest.NewRecorder()
	handler.Verify(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("verification status = %d, want 403", recorder.Code)
	}
}

func TestParseWebhookCollectsMessagesAndStatuses(t *testing.T) {
	body := []byte(`{
		"object":"whatsapp_business_account",
		"entry":[{"id":"waba-local","changes":[{"field":"messages","value":{
			"messaging_product":"whatsapp",
			"metadata":{"display_phone_number":"+50370000000","phone_number_id":"phone-local"},
			"contacts":[{"wa_id":"50370123456","profile":{"name":"Ana Martínez"}}],
			"messages":[{"from":"50370123456","id":"wamid.inbound.1","timestamp":"1787169600","type":"text","text":{"body":"Necesito una cotización"}}],
			"statuses":[{"id":"wamid.outbound.1","recipient_id":"50370123456","status":"delivered","timestamp":"1787169660"}]
		}}]}]
	}`)
	events, err := ParseWebhook(body)
	if err != nil {
		t.Fatalf("ParseWebhook returned an error: %v", err)
	}
	if len(events.Inbound) != 1 || events.Inbound[0].Text != "Necesito una cotización" || events.Inbound[0].ContactName != "Ana Martínez" {
		t.Fatalf("unexpected inbound events: %#v", events.Inbound)
	}
	if len(events.Statuses) != 1 || events.Statuses[0].Status != "delivered" {
		t.Fatalf("unexpected status events: %#v", events.Statuses)
	}
}

func TestReceiveRequiresValidSignatureBeforeProcessing(t *testing.T) {
	processor := &recordingProcessor{}
	handler := NewWebhookHandler("verify-me", "local-secret", processor)
	body := []byte(`{"object":"whatsapp_business_account","entry":[]}`)

	request := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	request.Header.Set("X-Hub-Signature-256", SignForTest(body, "wrong-secret"))
	recorder := httptest.NewRecorder()
	handler.Receive(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature status = %d, want 401", recorder.Code)
	}
	if processor.events.Inbound != nil {
		t.Fatal("processor was called for an invalid signature")
	}

	request = httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	request.Header.Set("X-Hub-Signature-256", SignForTest(body, "local-secret"))
	recorder = httptest.NewRecorder()
	handler.Receive(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("valid signature status = %d, body %q", recorder.Code, recorder.Body.String())
	}
	var result ProcessResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
}
