package webchat

import (
	"errors"
	"time"
)

const (
	SessionTokenHeader     = "X-RentStage-Chat-Token"
	SessionDuration        = 7 * 24 * time.Hour
	MaximumMessageLength   = 2000
	MaximumMessagesPerHour = 60
)

var (
	ErrNotFound       = errors.New("web chat not found")
	ErrDisabled       = errors.New("web chat is disabled")
	ErrInvalidToken   = errors.New("web chat token is invalid")
	ErrSessionClosed  = errors.New("web chat session is closed")
	ErrSessionExpired = errors.New("web chat session is expired")
	ErrRateLimited    = errors.New("web chat message rate exceeded")
)

type CreateSessionInput struct {
	ContactName     string  `json:"contact_name"`
	ContactEmail    *string `json:"contact_email"`
	Message         string  `json:"message"`
	ClientMessageID string  `json:"client_message_id"`
	ConsentAccepted bool    `json:"consent_accepted"`
	Website         string  `json:"website"`
}

type SendMessageInput struct {
	Body            string `json:"body"`
	ClientMessageID string `json:"client_message_id"`
}

type PublicMessage struct {
	ID         string    `json:"id"`
	Direction  string    `json:"direction"`
	SenderType string    `json:"sender_type"`
	Body       string    `json:"body"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type SessionView struct {
	ID          string          `json:"id"`
	Status      string          `json:"status"`
	ContactName string          `json:"contact_name"`
	ExpiresAt   time.Time       `json:"expires_at"`
	Messages    []PublicMessage `json:"messages"`
}

type CreateSessionResult struct {
	Session SessionView `json:"session"`
	Token   string      `json:"token"`
}

type publicConfiguration struct {
	TenantID     string
	TenantName   string
	TenantSlug   string
	TermsVersion string
}

type normalizedCreateSession struct {
	ContactName     string
	ContactEmail    *string
	Message         string
	ClientMessageID string
	TermsVersion    string
}

type normalizedMessage struct {
	Body            string
	ClientMessageID string
}
