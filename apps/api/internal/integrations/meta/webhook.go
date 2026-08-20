package meta

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const signaturePrefix = "sha256="

var (
	ErrInvalidSignature = errors.New("invalid Meta webhook signature")
	ErrInvalidPayload   = errors.New("invalid Meta webhook payload")
)

type InboundMessage struct {
	WABAID        string
	PhoneNumberID string
	DisplayPhone  string
	MessageID     string
	From          string
	ContactName   string
	Type          string
	Text          string
	OccurredAt    time.Time
}

type StatusUpdate struct {
	WABAID        string
	PhoneNumberID string
	MessageID     string
	RecipientID   string
	Status        string
	OccurredAt    time.Time
	Errors        []WebhookError
}

type WebhookError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type WebhookEvents struct {
	Inbound  []InboundMessage
	Statuses []StatusUpdate
}

type ProcessResult struct {
	InboundProcessed  int `json:"inbound_processed"`
	StatusesProcessed int `json:"statuses_processed"`
	Duplicates        int `json:"duplicates"`
	Ignored           int `json:"ignored"`
}

type Processor interface {
	ProcessMetaWebhook(context.Context, WebhookEvents) (ProcessResult, error)
}

type WebhookHandler struct {
	verifyToken string
	appSecret   string
	processor   Processor
}

func NewWebhookHandler(verifyToken, appSecret string, processor Processor) *WebhookHandler {
	return &WebhookHandler{
		verifyToken: strings.TrimSpace(verifyToken),
		appSecret:   strings.TrimSpace(appSecret),
		processor:   processor,
	}
}

func (h *WebhookHandler) Verify(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	challenge := query.Get("hub.challenge")
	if query.Get("hub.mode") != "subscribe" ||
		!constantTimeTextEqual(query.Get("hub.verify_token"), h.verifyToken) ||
		!validVerificationChallenge(challenge) {
		http.Error(w, "webhook verification failed", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(challenge)) // #nosec G705 -- Meta requires the exact allowlisted challenge in the response body.
}

func validVerificationChallenge(value string) bool {
	if value == "" || len(value) > 256 {
		return false
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-', character == '_', character == '.', character == '~':
		default:
			return false
		}
	}
	return true
}

func (h *WebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	raw, err := io.ReadAll(r.Body)
	if err != nil || len(raw) == 0 {
		http.Error(w, "invalid webhook body", http.StatusBadRequest)
		return
	}
	if err := ValidateSignature(raw, r.Header.Get("X-Hub-Signature-256"), h.appSecret); err != nil {
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}
	events, err := ParseWebhook(raw)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	result, err := h.processor.ProcessMetaWebhook(r.Context(), events)
	if err != nil {
		http.Error(w, "webhook processing failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func ValidateSignature(body []byte, header, appSecret string) error {
	if strings.TrimSpace(appSecret) == "" || !strings.HasPrefix(header, signaturePrefix) {
		return ErrInvalidSignature
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(header, signaturePrefix))
	if err != nil {
		return ErrInvalidSignature
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write(body)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return ErrInvalidSignature
	}
	return nil
}

func SignForTest(body []byte, appSecret string) string {
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write(body)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

func ParseWebhook(body []byte) (WebhookEvents, error) {
	var envelope webhookEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return WebhookEvents{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if envelope.Object != "whatsapp_business_account" {
		return WebhookEvents{}, fmt.Errorf("%w: unexpected object", ErrInvalidPayload)
	}
	result := WebhookEvents{Inbound: []InboundMessage{}, Statuses: []StatusUpdate{}}
	for _, entry := range envelope.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			contacts := make(map[string]string, len(change.Value.Contacts))
			for _, contact := range change.Value.Contacts {
				contacts[contact.WAID] = strings.TrimSpace(contact.Profile.Name)
			}
			for _, message := range change.Value.Messages {
				text := ""
				if message.Type == "text" {
					text = strings.TrimSpace(message.Text.Body)
				}
				result.Inbound = append(result.Inbound, InboundMessage{
					WABAID: entry.ID, PhoneNumberID: change.Value.Metadata.PhoneNumberID,
					DisplayPhone: change.Value.Metadata.DisplayPhoneNumber,
					MessageID:    message.ID, From: message.From,
					ContactName: contacts[message.From], Type: message.Type,
					Text: text, OccurredAt: unixTime(message.Timestamp),
				})
			}
			for _, status := range change.Value.Statuses {
				result.Statuses = append(result.Statuses, StatusUpdate{
					WABAID: entry.ID, PhoneNumberID: change.Value.Metadata.PhoneNumberID,
					MessageID: status.ID, RecipientID: status.RecipientID,
					Status: strings.ToLower(status.Status), OccurredAt: unixTime(status.Timestamp),
					Errors: status.Errors,
				})
			}
		}
	}
	return result, nil
}

func unixTime(value string) time.Time {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 {
		return time.Now().UTC()
	}
	return time.Unix(seconds, 0).UTC()
}

func constantTimeTextEqual(left, right string) bool {
	return hmac.Equal([]byte(left), []byte(right))
}

type webhookEnvelope struct {
	Object string         `json:"object"`
	Entry  []webhookEntry `json:"entry"`
}

type webhookEntry struct {
	ID      string          `json:"id"`
	Changes []webhookChange `json:"changes"`
}

type webhookChange struct {
	Field string       `json:"field"`
	Value webhookValue `json:"value"`
}

type webhookValue struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         webhookMetadata  `json:"metadata"`
	Contacts         []webhookContact `json:"contacts"`
	Messages         []webhookMessage `json:"messages"`
	Statuses         []webhookStatus  `json:"statuses"`
}

type webhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type webhookContact struct {
	WAID    string `json:"wa_id"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
}

type webhookMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text"`
}

type webhookStatus struct {
	ID          string         `json:"id"`
	RecipientID string         `json:"recipient_id"`
	Status      string         `json:"status"`
	Timestamp   string         `json:"timestamp"`
	Errors      []WebhookError `json:"errors"`
}
