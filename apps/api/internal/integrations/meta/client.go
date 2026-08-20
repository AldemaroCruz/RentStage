package meta

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrDeliveryFailed = errors.New("Meta message delivery failed")

type Client struct {
	baseURL       string
	graphVersion  string
	phoneNumberID string
	accessToken   string
	httpClient    *http.Client
}

func NewClient(baseURL, graphVersion, phoneNumberID, accessToken string) *Client {
	return &Client{
		baseURL:       strings.TrimRight(baseURL, "/"),
		graphVersion:  strings.Trim(graphVersion, "/"),
		phoneNumberID: strings.TrimSpace(phoneNumberID),
		accessToken:   strings.TrimSpace(accessToken),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SendText(ctx context.Context, to, body string) (string, error) {
	recipient := strings.TrimPrefix(strings.TrimSpace(to), "+")
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                recipient,
		"type":              "text",
		"text":              map[string]any{"preview_url": false, "body": strings.TrimSpace(body)},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf(
		"%s/%s/%s/messages",
		c.baseURL,
		url.PathEscape(c.graphVersion),
		url.PathEscape(c.phoneNumberID),
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+c.accessToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDeliveryFailed, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("%w: read response", ErrDeliveryFailed)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("%w: HTTP %d", ErrDeliveryFailed, response.StatusCode)
	}
	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil || len(result.Messages) == 0 || result.Messages[0].ID == "" {
		return "", fmt.Errorf("%w: missing message id", ErrDeliveryFailed)
	}
	return result.Messages[0].ID, nil
}

type LocalGraphHandler struct {
	accessToken   string
	graphVersion  string
	phoneNumberID string
}

func NewLocalGraphHandler(accessToken, graphVersion, phoneNumberID string) *LocalGraphHandler {
	return &LocalGraphHandler{
		accessToken:   accessToken,
		graphVersion:  graphVersion,
		phoneNumberID: phoneNumberID,
	}
}

func (h *LocalGraphHandler) Send(w http.ResponseWriter, r *http.Request) {
	if !constantTimeTextEqual(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), h.accessToken) ||
		r.PathValue("graphVersion") != h.graphVersion ||
		r.PathValue("phoneNumberID") != h.phoneNumberID {
		http.Error(w, "local Graph credentials rejected", http.StatusUnauthorized)
		return
	}
	var payload struct {
		MessagingProduct string `json:"messaging_product"`
		To               string `json:"to"`
		Type             string `json:"type"`
		Text             struct {
			Body string `json:"body"`
		} `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil ||
		payload.MessagingProduct != "whatsapp" || payload.Type != "text" ||
		strings.TrimSpace(payload.To) == "" || strings.TrimSpace(payload.Text.Body) == "" {
		http.Error(w, "invalid local Graph request", http.StatusBadRequest)
		return
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		http.Error(w, "could not create local message id", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"messaging_product": "whatsapp",
		"contacts":          []map[string]string{{"input": payload.To, "wa_id": payload.To}},
		"messages":          []map[string]string{{"id": "wamid.local." + hex.EncodeToString(random)}},
	})
}
