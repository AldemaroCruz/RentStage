package meta

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendTextUsesCloudAPIContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v-test/phone-123/messages" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-local" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["messaging_product"] != "whatsapp" || payload["to"] != "50370123456" || payload["type"] != "text" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.outbound.1"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "v-test", "phone-123", "token-local")
	messageID, err := client.SendText(context.Background(), "+50370123456", "Hola desde RentStage")
	if err != nil {
		t.Fatalf("SendText returned an error: %v", err)
	}
	if messageID != "wamid.outbound.1" {
		t.Fatalf("message ID = %q", messageID)
	}
}

func TestSendTextRejectsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rejected", http.StatusBadRequest)
	}))
	defer server.Close()
	client := NewClient(server.URL, "v-test", "phone-123", "token-local")
	if _, err := client.SendText(context.Background(), "50370123456", "Hola"); err == nil {
		t.Fatal("expected provider failure")
	}
}

func TestLocalGraphHandlerRequiresIsolatedContract(t *testing.T) {
	handler := NewLocalGraphHandler("token-local", "v-test", "phone-123")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{graphVersion}/{phoneNumberID}/messages", handler.Send)

	body := `{"messaging_product":"whatsapp","to":"50370123456","type":"text","text":{"body":"Hola"}}`
	request := httptest.NewRequest(http.MethodPost, "/v-test/phone-123/messages", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token-local")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("local Graph response = %d %q", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/v-other/phone-123/messages", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token-local")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("mismatched version response = %d, want 401", recorder.Code)
	}
}
