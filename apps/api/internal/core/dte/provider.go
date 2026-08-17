package dte

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Provider interface {
	Submit(context.Context, ProviderSubmission) (ProviderResult, error)
	Invalidate(context.Context, ProviderSubmission, string) (InvalidationResult, error)
}

func providerFor(settings Settings) (Provider, error) {
	switch settings.ProviderMode {
	case "MOCK":
		return MockProvider{}, nil
	case "MH_HTTP":
		return NewMHHTTPProvider(settings)
	default:
		return nil, ErrProviderConfiguration
	}
}

type MockProvider struct{}

func (MockProvider) Submit(_ context.Context, submission ProviderSubmission) (ProviderResult, error) {
	encoded, err := json.Marshal(submission.Document.Payload)
	if err != nil {
		return ProviderResult{}, fmt.Errorf("encode mock dte: %w", err)
	}
	if len(encoded) < 80 {
		return ProviderResult{
			Accepted:       false,
			ErrorCode:      "MOCK_SCHEMA_REJECTED",
			ErrorMessage:   "El documento simulado no contiene suficiente información.",
			ProviderStatus: "RECHAZADO",
			Request:        map[string]any{"provider": "MOCK", "bytes": len(encoded)},
			Response:       map[string]any{"estado": "RECHAZADO"},
		}, nil
	}
	digest := sha256.Sum256(encoded)
	signed := "MOCK-SIGNED." + hex.EncodeToString(digest[:]) + "." + string(encoded)
	sealDigest := sha256.Sum256([]byte(submission.Document.GenerationCode + submission.Document.ControlNumber + hex.EncodeToString(digest[:])))
	seal := "MOCK-" + strings.ToUpper(hex.EncodeToString(sealDigest[:16]))
	return ProviderResult{
		Accepted:       true,
		SignedDocument: signed,
		ReceiptSeal:    seal,
		ProviderStatus: "PROCESADO",
		Request: map[string]any{
			"provider":        "MOCK",
			"environment":     submission.Settings.Environment,
			"generation_code": submission.Document.GenerationCode,
		},
		Response: map[string]any{
			"estado":         "PROCESADO",
			"selloRecibido":  seal,
			"descripcionMsg": "Aceptado por el proveedor local de certificación.",
		},
	}, nil
}

func (MockProvider) Invalidate(_ context.Context, submission ProviderSubmission, reason string) (InvalidationResult, error) {
	if strings.TrimSpace(submission.Document.ReceiptSeal) == "" {
		return InvalidationResult{}, ErrInvalidationUnsupported
	}
	digest := sha256.Sum256([]byte(submission.Document.ReceiptSeal + reason))
	return InvalidationResult{
		Accepted:       true,
		ProviderStatus: "INVALIDADO",
		Request: map[string]any{
			"provider":        "MOCK",
			"generation_code": submission.Document.GenerationCode,
			"reason":          reason,
		},
		Response: map[string]any{
			"estado":            "INVALIDADO",
			"selloInvalidacion": "MOCK-INV-" + strings.ToUpper(hex.EncodeToString(digest[:12])),
		},
	}, nil
}

type SecretResolver interface {
	Resolve(context.Context, string) (string, error)
}

type EnvSecretResolver struct{}

func (EnvSecretResolver) Resolve(_ context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if !strings.HasPrefix(reference, "env://") {
		return "", fmt.Errorf("only env:// secret references are supported by this runtime")
	}
	key := strings.TrimSpace(strings.TrimPrefix(reference, "env://"))
	if key == "" {
		return "", fmt.Errorf("environment secret reference is empty")
	}
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("secret environment variable %s is empty", key)
	}
	return value, nil
}

type MHHTTPProvider struct {
	settings Settings
	client   *http.Client
	secrets  SecretResolver
}

func NewMHHTTPProvider(settings Settings) (*MHHTTPProvider, error) {
	for _, endpoint := range []string{settings.AuthURL, settings.SignerURL, settings.ReceptionURL} {
		if err := validateProviderEndpoint(endpoint, settings.Environment == "PRODUCTION"); err != nil {
			return nil, err
		}
	}
	for _, endpoint := range []string{settings.InvalidationURL, settings.QueryURL} {
		if strings.TrimSpace(endpoint) == "" {
			continue
		}
		if err := validateProviderEndpoint(endpoint, settings.Environment == "PRODUCTION"); err != nil {
			return nil, err
		}
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           safeExternalDialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many DTE provider redirects: %w", ErrProviderConfiguration)
			}
			return validateProviderEndpoint(req.URL.String(), settings.Environment == "PRODUCTION")
		},
	}
	return &MHHTTPProvider{
		settings: settings,
		client:   client,
		secrets:  EnvSecretResolver{},
	}, nil
}

func validateProviderEndpoint(endpoint string, requireHTTPS bool) error {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Hostname() == "" {
		return ErrProviderConfiguration
	}
	if parsed.User != nil {
		return fmt.Errorf("DTE endpoint credentials in URLs are not allowed: %w", ErrProviderConfiguration)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrProviderConfiguration
	}
	if requireHTTPS && parsed.Scheme != "https" {
		return fmt.Errorf("production DTE endpoints must use HTTPS: %w", ErrProviderConfiguration)
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("local DTE endpoints are not allowed: %w", ErrProviderConfiguration)
	}
	if ip := net.ParseIP(host); ip != nil && unsafeProviderIP(ip) {
		return fmt.Errorf("private or link-local DTE endpoints are not allowed: %w", ErrProviderConfiguration)
	}
	return nil
}

func safeExternalDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid DTE provider address: %w", ErrProviderConfiguration)
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return nil, fmt.Errorf("resolve DTE provider host: %w", ErrProviderUnavailable)
	}
	for _, addressItem := range addresses {
		if unsafeProviderIP(addressItem.IP) {
			return nil, fmt.Errorf("DTE provider resolved to a private or link-local address: %w", ErrProviderConfiguration)
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	var lastErr error
	for _, addressItem := range addresses {
		connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(addressItem.IP.String(), port))
		if dialErr == nil {
			return connection, nil
		}
		lastErr = dialErr
	}
	return nil, fmt.Errorf("connect to DTE provider: %v: %w", lastErr, ErrProviderUnavailable)
}

func unsafeProviderIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if value := ip.To4(); value != nil {
		// Carrier-grade NAT, benchmarking, and the cloud metadata/link-local range
		// must never be reachable through a tenant-configurable fiscal adapter.
		if value[0] == 100 && value[1] >= 64 && value[1] <= 127 {
			return true
		}
		if value[0] == 198 && (value[1] == 18 || value[1] == 19) {
			return true
		}
	}
	return false
}

func (p *MHHTTPProvider) Submit(ctx context.Context, submission ProviderSubmission) (ProviderResult, error) {
	user, err := p.secrets.Resolve(ctx, p.settings.UserSecretRef)
	if err != nil {
		return ProviderResult{}, fmt.Errorf("resolve DTE user: %w", err)
	}
	password, err := p.secrets.Resolve(ctx, p.settings.PasswordSecretRef)
	if err != nil {
		return ProviderResult{}, fmt.Errorf("resolve DTE password: %w", err)
	}
	signingPassword, err := p.secrets.Resolve(ctx, p.settings.SigningPasswordSecretRef)
	if err != nil {
		return ProviderResult{}, fmt.Errorf("resolve DTE signing password: %w", err)
	}

	token, authPayload, err := p.authenticate(ctx, user, password)
	if err != nil {
		return ProviderResult{}, err
	}
	signed, signerPayload, err := p.sign(ctx, submission.Document.Payload, signingPassword)
	if err != nil {
		return ProviderResult{}, err
	}
	result, receptionPayload, err := p.transmit(ctx, token, submission.Document, signed)
	if err != nil {
		return ProviderResult{}, err
	}
	result.SignedDocument = signed
	result.Request = map[string]any{
		"authentication": redactMap(authPayload),
		"signer":         redactMap(signerPayload),
		"reception": map[string]any{
			"generation_code": submission.Document.GenerationCode,
			"control_number":  submission.Document.ControlNumber,
			"document_type":   submission.Document.DocumentType,
		},
	}
	result.Response = redactMap(receptionPayload)
	return result, nil
}

func (p *MHHTTPProvider) Invalidate(ctx context.Context, submission ProviderSubmission, reason string) (InvalidationResult, error) {
	if strings.TrimSpace(p.settings.InvalidationURL) == "" {
		return InvalidationResult{}, ErrInvalidationUnsupported
	}
	user, err := p.secrets.Resolve(ctx, p.settings.UserSecretRef)
	if err != nil {
		return InvalidationResult{}, err
	}
	password, err := p.secrets.Resolve(ctx, p.settings.PasswordSecretRef)
	if err != nil {
		return InvalidationResult{}, err
	}
	token, _, err := p.authenticate(ctx, user, password)
	if err != nil {
		return InvalidationResult{}, err
	}
	request := map[string]any{
		"ambiente":         environmentCode(p.settings.Environment),
		"codigoGeneracion": strings.ToUpper(submission.Document.GenerationCode),
		"numeroControl":    submission.Document.ControlNumber,
		"selloRecibido":    submission.Document.ReceiptSeal,
		"motivo":           reason,
	}
	response, status, err := p.doJSON(ctx, http.MethodPost, p.settings.InvalidationURL, token, request)
	if err != nil {
		return InvalidationResult{}, err
	}
	providerStatus := firstString(response, "estado", "status", "resultado")
	accepted := status >= 200 && status < 300 && !looksRejected(providerStatus, response)
	result := InvalidationResult{
		Accepted:       accepted,
		Retryable:      status == http.StatusTooManyRequests || status >= 500,
		ProviderStatus: providerStatus,
		Request:        request,
		Response:       redactMap(response),
	}
	if !accepted {
		result.ErrorCode = firstString(response, "codigoMsg", "codigo", "error")
		result.ErrorMessage = firstString(response, "descripcionMsg", "mensaje", "message")
	}
	return result, nil
}

func (p *MHHTTPProvider) authenticate(ctx context.Context, user, password string) (string, map[string]any, error) {
	request := map[string]any{"user": user, "pwd": password}
	response, status, err := p.doJSON(ctx, http.MethodPost, p.settings.AuthURL, "", request)
	if err != nil {
		return "", request, fmt.Errorf("authenticate with DTE provider: %w", err)
	}
	if status < 200 || status >= 300 {
		return "", request, fmt.Errorf("DTE authentication returned HTTP %d: %w", status, ErrProviderUnavailable)
	}
	token := firstString(response, "token", "body", "access_token")
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		return "", request, fmt.Errorf("DTE authentication did not return a token: %w", ErrProviderUnavailable)
	}
	return token, request, nil
}

func (p *MHHTTPProvider) sign(ctx context.Context, payload map[string]any, password string) (string, map[string]any, error) {
	request := map[string]any{
		"nit":         strings.TrimSpace(fmt.Sprint(payloadPath(payload, "emisor", "nit"))),
		"activo":      true,
		"passwordPri": password,
		"dteJson":     payload,
	}
	response, status, err := p.doJSON(ctx, http.MethodPost, p.settings.SignerURL, "", request)
	if err != nil {
		return "", request, fmt.Errorf("sign DTE: %w", err)
	}
	if status < 200 || status >= 300 {
		return "", request, fmt.Errorf("DTE signer returned HTTP %d: %w", status, ErrProviderUnavailable)
	}
	signed := firstString(response, "body", "documento", "signedDocument")
	if signed == "" {
		return "", request, fmt.Errorf("DTE signer did not return a signed document: %w", ErrProviderUnavailable)
	}
	return signed, request, nil
}

func (p *MHHTTPProvider) transmit(ctx context.Context, token string, document DocumentDetail, signed string) (ProviderResult, map[string]any, error) {
	sequence := sequenceFromControl(document.ControlNumber)
	request := map[string]any{
		"ambiente":         environmentCode(p.settings.Environment),
		"idEnvio":          sequence,
		"version":          document.SchemaVersion,
		"tipoDte":          document.DocumentType,
		"documento":        signed,
		"codigoGeneracion": strings.ToUpper(document.GenerationCode),
	}
	response, status, err := p.doJSON(ctx, http.MethodPost, p.settings.ReceptionURL, token, request)
	if err != nil {
		return ProviderResult{}, request, fmt.Errorf("transmit DTE: %w", err)
	}
	providerStatus := firstString(response, "estado", "status", "resultado")
	accepted := status >= 200 && status < 300 && !looksRejected(providerStatus, response)
	result := ProviderResult{
		Accepted:       accepted,
		Retryable:      status == http.StatusTooManyRequests || status >= 500,
		ReceiptSeal:    firstString(response, "selloRecibido", "selloRecepcion", "receiptSeal"),
		ProviderStatus: providerStatus,
	}
	if accepted && result.ReceiptSeal == "" {
		result.Accepted = false
		result.ErrorCode = "MISSING_RECEIPT_SEAL"
		result.ErrorMessage = "El proveedor respondió sin sello de recepción."
	}
	if !result.Accepted {
		result.ErrorCode = firstNonEmpty(result.ErrorCode, firstString(response, "codigoMsg", "codigo", "error"))
		result.ErrorMessage = firstNonEmpty(result.ErrorMessage, firstString(response, "descripcionMsg", "mensaje", "message"))
	}
	return result, request, nil
}

func (p *MHHTTPProvider) doJSON(ctx context.Context, method, endpoint, token string, input any) (map[string]any, int, error) {
	if err := validateProviderEndpoint(
		endpoint,
		p.settings.Environment == "PRODUCTION",
	); err != nil {
		return nil, 0, err
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return nil, 0, err
	}
	// #nosec G704 -- The endpoint is validated immediately above, while the
	// custom transport rejects private, loopback, link-local and rebinding
	// destinations and redirects are revalidated.
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// #nosec G704 -- The request URL passed provider validation and the custom
	// transport only connects to validated global-unicast destinations.
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, ErrProviderUnavailable
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	payload := map[string]any{}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &payload); err != nil {
			payload["raw"] = string(data)
		}
	}
	return payload, resp.StatusCode, nil
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case map[string]any:
			if nested := firstString(typed, "token", "body", "value", "message"); nested != "" {
				return nested
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func looksRejected(status string, payload map[string]any) bool {
	value := strings.ToUpper(status + " " + firstString(payload, "descripcionMsg", "mensaje", "message"))
	return strings.Contains(value, "RECHAZ") || strings.Contains(value, "ERROR") || strings.Contains(value, "INVALID")
}

func redactMap(input map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range input {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || lower == "pwd" || strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
			result[key] = "***"
			continue
		}
		result[key] = redactValue(value)
	}
	return result
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return redactMap(typed)
	case []any:
		items := make([]any, len(typed))
		for index, item := range typed {
			items[index] = redactValue(item)
		}
		return items
	default:
		return value
	}
}

func payloadPath(payload map[string]any, keys ...string) any {
	var current any = payload
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[key]
	}
	return current
}
