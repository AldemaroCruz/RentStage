package webchat

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestVisibleDraftConversationQueryEnforcesVisibilityBoundary(
	t *testing.T,
) {
	requiredClauses := []string{
		"message.tenant_id = $1",
		"message.conversation_id = $2",
		"message.provider = 'WEB_CHAT'",
		"message.sender_type = 'CUSTOMER'",
		"message.status = 'RECEIVED'",
		"message.sender_type = 'USER'",
		"'SENT'",
		"'DELIVERED'",
		"'READ'",
		"LIMIT $3",
	}

	for _, clause := range requiredClauses {
		if !strings.Contains(
			visibleDraftConversationQuery,
			clause,
		) {
			t.Fatalf("visible context query is missing %q", clause)
		}
	}

	for _, hiddenStatus := range []string{
		"'DRAFT'",
		"'APPROVED'",
		"'FAILED'",
	} {
		if strings.Contains(
			visibleDraftConversationQuery,
			hiddenStatus,
		) {
			t.Fatalf(
				"visible context query exposes status %s",
				hiddenStatus,
			)
		}
	}
}

func TestDraftConversationMessageMapsVisibleDirections(
	t *testing.T,
) {
	tests := []struct {
		direction string
		wantRole  DraftMessageRole
	}{
		{
			direction: "INBOUND",
			wantRole:  DraftMessageRoleCustomer,
		},
		{
			direction: "OUTBOUND",
			wantRole:  DraftMessageRoleTeam,
		},
	}

	for _, test := range tests {
		t.Run(test.direction, func(t *testing.T) {
			message, err := draftConversationMessage(
				test.direction,
				"  Mensaje visible  ",
			)
			if err != nil {
				t.Fatalf("map visible message: %v", err)
			}
			if message.Role != test.wantRole {
				t.Fatalf(
					"message role = %q, want %q",
					message.Role,
					test.wantRole,
				)
			}
			if message.Body != "Mensaje visible" {
				t.Fatalf("unexpected body: %q", message.Body)
			}
		})
	}
}

func TestDraftConversationMessageRejectsUnknownDirection(
	t *testing.T,
) {
	_, err := draftConversationMessage(
		"INTERNAL",
		"No debe llegar al modelo",
	)
	if !errors.Is(err, ErrInvalidDraft) {
		t.Fatalf("expected invalid draft error, got %v", err)
	}
}

func TestBoundedDraftConversationKeepsNewestMessagesInOrder(
	t *testing.T,
) {
	messages := make([]DraftConversationMessage, 0, 15)
	for index := 0; index < 15; index++ {
		messages = append(messages, DraftConversationMessage{
			Role: DraftMessageRoleCustomer,
			Body: fmt.Sprintf("message-%02d", index),
		})
	}

	bounded, err := boundedDraftConversation(messages)
	if err != nil {
		t.Fatalf("bound conversation: %v", err)
	}
	if len(bounded) != MaximumDraftContextMessages {
		t.Fatalf("unexpected message count: %d", len(bounded))
	}
	if bounded[0].Body != "message-03" {
		t.Fatalf("unexpected first message: %q", bounded[0].Body)
	}
	if bounded[len(bounded)-1].Body != "message-14" {
		t.Fatalf(
			"unexpected last message: %q",
			bounded[len(bounded)-1].Body,
		)
	}
}

func TestBoundedDraftConversationHonorsRuneBudget(
	t *testing.T,
) {
	messages := []DraftConversationMessage{
		{
			Role: DraftMessageRoleCustomer,
			Body: strings.Repeat("a", MaximumMessageLength),
		},
		{
			Role: DraftMessageRoleTeam,
			Body: strings.Repeat("b", MaximumMessageLength),
		},
		{
			Role: DraftMessageRoleCustomer,
			Body: strings.Repeat("c", MaximumMessageLength),
		},
		{
			Role: DraftMessageRoleTeam,
			Body: strings.Repeat("d", MaximumMessageLength),
		},
		{
			Role: DraftMessageRoleCustomer,
			Body: strings.Repeat("e", MaximumMessageLength),
		},
	}

	bounded, err := boundedDraftConversation(messages)
	if err != nil {
		t.Fatalf("bound conversation: %v", err)
	}
	if len(bounded) != 4 {
		t.Fatalf("unexpected message count: %d", len(bounded))
	}
	if bounded[0].Body[0] != 'b' {
		t.Fatalf("unexpected oldest retained message")
	}
	if bounded[len(bounded)-1].Body[0] != 'e' {
		t.Fatalf("unexpected newest retained message")
	}
}

func TestBoundedDraftConversationRejectsInvalidRetainedMessage(
	t *testing.T,
) {
	_, err := boundedDraftConversation(
		[]DraftConversationMessage{
			{Role: "SYSTEM", Body: "No debe llegar al modelo"},
		},
	)
	if !errors.Is(err, ErrInvalidDraft) {
		t.Fatalf("expected invalid draft error, got %v", err)
	}
}
