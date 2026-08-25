package webchat

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

func normalizeCreateSession(
	input CreateSessionInput,
	configuration publicConfiguration,
) (normalizedCreateSession, map[string]string) {
	result := normalizedCreateSession{
		ContactName:     strings.Join(strings.Fields(input.ContactName), " "),
		Message:         strings.TrimSpace(input.Message),
		ClientMessageID: strings.TrimSpace(input.ClientMessageID),
		TermsVersion:    strings.TrimSpace(configuration.TermsVersion),
	}
	fields := map[string]string{}

	nameLength := utf8.RuneCountInString(result.ContactName)
	if nameLength < 2 || nameLength > 120 {
		fields["contact_name"] = "Name must contain between 2 and 120 characters."
	}

	if input.ContactEmail != nil {
		email := strings.ToLower(strings.TrimSpace(*input.ContactEmail))
		if email != "" {
			result.ContactEmail = &email
			if len(email) > 320 || !validWebChatEmail(email) {
				fields["contact_email"] = "Email address is invalid."
			}
		}
	}

	messageLength := utf8.RuneCountInString(result.Message)
	if messageLength < 1 || messageLength > MaximumMessageLength {
		fields["message"] = "Message must contain between 1 and 2,000 characters."
	}
	if !idutil.IsUUID(result.ClientMessageID) {
		fields["client_message_id"] = "Client message ID is invalid."
	}
	if !input.ConsentAccepted {
		fields["consent_accepted"] = "Accept the contact and privacy notice to continue."
	}
	if strings.TrimSpace(input.Website) != "" {
		fields["website"] = "Leave this field empty."
	}

	return result, fields
}

func normalizeSendMessage(input SendMessageInput) (normalizedMessage, map[string]string) {
	result := normalizedMessage{
		Body:            strings.TrimSpace(input.Body),
		ClientMessageID: strings.TrimSpace(input.ClientMessageID),
	}
	fields := map[string]string{}

	messageLength := utf8.RuneCountInString(result.Body)
	if messageLength < 1 || messageLength > MaximumMessageLength {
		fields["body"] = "Message must contain between 1 and 2,000 characters."
	}
	if !idutil.IsUUID(result.ClientMessageID) {
		fields["client_message_id"] = "Client message ID is invalid."
	}

	return result, fields
}

func validWebChatEmail(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil &&
		parsed.Name == "" &&
		strings.EqualFold(parsed.Address, value)
}
