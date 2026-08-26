package webchat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const visibleDraftConversationQuery = `
	SELECT recent.direction, recent.body
	FROM (
		SELECT
			message.id,
			message.direction,
			message.body,
			message.created_at
		FROM assistant_messages message
		WHERE message.tenant_id = $1
		  AND message.conversation_id = $2
		  AND message.provider = 'WEB_CHAT'
		  AND (
			(
				message.direction = 'INBOUND'
				AND message.sender_type = 'CUSTOMER'
				AND message.status = 'RECEIVED'
			)
			OR (
				message.direction = 'OUTBOUND'
				AND message.sender_type = 'USER'
				AND message.status IN (
					'SENT',
					'DELIVERED',
					'READ'
				)
			)
		  )
		ORDER BY message.created_at DESC, message.id DESC
		LIMIT $3
	) recent
	ORDER BY recent.created_at, recent.id
`

type inboundDraftPreparation struct {
	Request   DraftRequest
	Duplicate bool
}

func (r *Repository) PrepareInboundDraft(
	ctx context.Context,
	tenantSlug string,
	sessionID string,
	tokenHash string,
	input normalizedMessage,
	now time.Time,
) (inboundDraftPreparation, error) {
	var tenantID string
	var conversationID string
	var tenantName string
	var canonicalTenantSlug string
	var contactName string
	var status string
	var expiresAt time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT
			session.tenant_id,
			session.conversation_id,
			tenant.name,
			tenant.slug,
			conversation.contact_name,
			session.status,
			session.expires_at
		FROM assistant_web_chat_sessions session
		JOIN tenants tenant
		  ON tenant.id = session.tenant_id
		JOIN assistant_conversations conversation
		  ON conversation.tenant_id = session.tenant_id
		 AND conversation.id = session.conversation_id
		JOIN public_catalog_settings settings
		  ON settings.tenant_id = session.tenant_id
		WHERE tenant.slug = $1
		  AND tenant.status = 'ACTIVE'
		  AND settings.enabled = TRUE
		  AND settings.web_chat_enabled = TRUE
		  AND session.id = $2
		  AND session.token_hash = $3
	`,
		strings.ToLower(strings.TrimSpace(tenantSlug)),
		sessionID,
		tokenHash,
	).Scan(
		&tenantID,
		&conversationID,
		&tenantName,
		&canonicalTenantSlug,
		&contactName,
		&status,
		&expiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return inboundDraftPreparation{}, ErrInvalidToken
	}
	if err != nil {
		return inboundDraftPreparation{}, fmt.Errorf(
			"prepare inbound web chat draft: %w",
			err,
		)
	}
	if status != "ACTIVE" {
		return inboundDraftPreparation{}, ErrSessionClosed
	}
	if !now.Before(expiresAt) {
		return inboundDraftPreparation{}, ErrSessionExpired
	}

	var duplicate bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM assistant_messages
			WHERE tenant_id = $1
			  AND provider = 'WEB_CHAT'
			  AND external_message_id = $2
		)
	`,
		tenantID,
		input.ClientMessageID,
	).Scan(&duplicate); err != nil {
		return inboundDraftPreparation{}, fmt.Errorf(
			"check duplicate before web chat draft: %w",
			err,
		)
	}

	preparation := inboundDraftPreparation{
		Duplicate: duplicate,
		Request: DraftRequest{
			Kind:            DraftKindFollowUp,
			TenantName:      tenantName,
			TenantSlug:      canonicalTenantSlug,
			ContactName:     contactName,
			CustomerMessage: input.Body,
		},
	}
	if duplicate {
		return preparation, nil
	}

	var recentMessages int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM assistant_messages
		WHERE tenant_id = $1
		  AND conversation_id = $2
		  AND direction = 'INBOUND'
		  AND created_at >= $3
	`,
		tenantID,
		conversationID,
		now.Add(-time.Hour),
	).Scan(&recentMessages); err != nil {
		return inboundDraftPreparation{}, fmt.Errorf(
			"count messages before web chat draft: %w",
			err,
		)
	}
	if recentMessages >= MaximumMessagesPerHour {
		return inboundDraftPreparation{}, ErrRateLimited
	}

	previousMessages, err := r.visibleDraftConversation(
		ctx,
		tenantID,
		conversationID,
	)
	if err != nil {
		return inboundDraftPreparation{}, err
	}
	preparation.Request.PreviousMessages = previousMessages

	salesContext, err := r.LoadDraftSalesContext(
		ctx,
		tenantID,
	)
	if err != nil {
		return inboundDraftPreparation{}, err
	}
	preparation.Request.SalesContext = salesContext

	return preparation, nil
}

func (r *Repository) visibleDraftConversation(
	ctx context.Context,
	tenantID string,
	conversationID string,
) ([]DraftConversationMessage, error) {
	rows, err := r.pool.Query(
		ctx,
		visibleDraftConversationQuery,
		tenantID,
		conversationID,
		MaximumDraftContextMessages,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query visible web chat draft context: %w",
			err,
		)
	}
	defer rows.Close()

	messages := make(
		[]DraftConversationMessage,
		0,
		MaximumDraftContextMessages,
	)

	for rows.Next() {
		var direction string
		var body string

		if err := rows.Scan(&direction, &body); err != nil {
			return nil, fmt.Errorf(
				"scan visible web chat draft context: %w",
				err,
			)
		}

		message, err := draftConversationMessage(
			direction,
			body,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate visible web chat draft context: %w",
			err,
		)
	}

	messages, err = boundedDraftConversation(messages)
	if err != nil {
		return nil, fmt.Errorf(
			"bound visible web chat draft context: %w",
			err,
		)
	}

	return messages, nil
}

func draftConversationMessage(
	direction string,
	body string,
) (DraftConversationMessage, error) {
	message := DraftConversationMessage{
		Body: strings.TrimSpace(body),
	}

	switch direction {
	case "INBOUND":
		message.Role = DraftMessageRoleCustomer
	case "OUTBOUND":
		message.Role = DraftMessageRoleTeam
	default:
		return DraftConversationMessage{}, fmt.Errorf(
			"%w: unsupported visible message direction %q",
			ErrInvalidDraft,
			direction,
		)
	}

	return message, nil
}

func boundedDraftConversation(
	messages []DraftConversationMessage,
) ([]DraftConversationMessage, error) {
	if len(messages) > MaximumDraftContextMessages {
		messages = messages[len(messages)-MaximumDraftContextMessages:]
	}

	candidates := append(
		[]DraftConversationMessage(nil),
		messages...,
	)
	start := len(candidates)
	totalRunes := 0

	for index := len(candidates) - 1; index >= 0; index-- {
		candidates[index].Body = strings.TrimSpace(
			candidates[index].Body,
		)
		messageRunes := utf8.RuneCountInString(
			candidates[index].Body,
		)
		if totalRunes+messageRunes >
			MaximumDraftContextRunes {
			break
		}

		totalRunes += messageRunes
		start = index
	}

	return NormalizeDraftConversation(candidates[start:])
}

func (r *Repository) AddInboundMessage(
	ctx context.Context,
	tenantSlug string,
	sessionID string,
	tokenHash string,
	input normalizedMessage,
	responseDraft DraftResult,
	now time.Time,
) (SessionView, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SessionView{}, fmt.Errorf(
			"begin web chat message: %w",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tenantID string
	var conversationID string
	var status string
	var expiresAt time.Time

	err = tx.QueryRow(ctx, `
		SELECT
			session.tenant_id,
			session.conversation_id,
			session.status,
			session.expires_at
		FROM assistant_web_chat_sessions session
		JOIN tenants tenant
		  ON tenant.id = session.tenant_id
		JOIN public_catalog_settings settings
		  ON settings.tenant_id = session.tenant_id
		WHERE tenant.slug = $1
		  AND tenant.status = 'ACTIVE'
		  AND settings.enabled = TRUE
		  AND settings.web_chat_enabled = TRUE
		  AND session.id = $2
		  AND session.token_hash = $3
		FOR UPDATE OF session
	`,
		strings.ToLower(strings.TrimSpace(tenantSlug)),
		sessionID,
		tokenHash,
	).Scan(
		&tenantID,
		&conversationID,
		&status,
		&expiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionView{}, ErrInvalidToken
	}
	if err != nil {
		return SessionView{}, fmt.Errorf(
			"lock web chat session: %w",
			err,
		)
	}
	if status != "ACTIVE" {
		return SessionView{}, ErrSessionClosed
	}
	if !now.Before(expiresAt) {
		if _, err := tx.Exec(ctx, `
			UPDATE assistant_web_chat_sessions
			SET status = 'CLOSED',
			    updated_at = $2
			WHERE id = $1
		`, sessionID, now); err != nil {
			return SessionView{}, fmt.Errorf(
				"expire web chat session: %w",
				err,
			)
		}
		if err := tx.Commit(ctx); err != nil {
			return SessionView{}, fmt.Errorf(
				"commit web chat expiration: %w",
				err,
			)
		}
		return SessionView{}, ErrSessionExpired
	}

	var duplicate bool

	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM assistant_messages
			WHERE tenant_id = $1
			  AND provider = 'WEB_CHAT'
			  AND external_message_id = $2
		)
	`, tenantID, input.ClientMessageID).Scan(&duplicate); err != nil {
		return SessionView{}, fmt.Errorf(
			"check duplicate web chat message: %w",
			err,
		)
	}

	if !duplicate {
		var recentMessages int

		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)::int
			FROM assistant_messages
			WHERE tenant_id = $1
			  AND conversation_id = $2
			  AND direction = 'INBOUND'
			  AND created_at >= $3
		`,
			tenantID,
			conversationID,
			now.Add(-time.Hour),
		).Scan(&recentMessages); err != nil {
			return SessionView{}, fmt.Errorf(
				"count recent web chat messages: %w",
				err,
			)
		}

		if recentMessages >= MaximumMessagesPerHour {
			return SessionView{}, ErrRateLimited
		}

		var insertedMessageID string

		err = tx.QueryRow(ctx, `
			INSERT INTO assistant_messages (
				tenant_id,
				conversation_id,
				direction,
				sender_type,
				provider,
				external_message_id,
				body,
				status,
				metadata,
				created_at
			) VALUES (
				$1,
				$2,
				'INBOUND',
				'CUSTOMER',
				'WEB_CHAT',
				$3,
				$4,
				'RECEIVED',
				jsonb_build_object(
					'transport', 'PUBLIC_WEB_CHAT',
					'client_message_id', $3::varchar(180)
				),
				$5
			)
			ON CONFLICT (
				tenant_id,
				provider,
				external_message_id
			)
			  WHERE external_message_id IS NOT NULL
			DO NOTHING
			RETURNING id
		`,
			tenantID,
			conversationID,
			input.ClientMessageID,
			input.Body,
			now,
		).Scan(&insertedMessageID)

		if errors.Is(err, pgx.ErrNoRows) {
			duplicate = true
		} else if err != nil {
			return SessionView{}, fmt.Errorf(
				"insert web chat message: %w",
				err,
			)
		}
	}

	if !duplicate {
		if strings.TrimSpace(responseDraft.Body) != "" {
			groundingReferences, err := encodeDraftGroundingReferences(
				responseDraft.GroundingReferences,
			)
			if err != nil {
				return SessionView{}, err
			}
			salesBrief, err := encodeDraftSalesBrief(responseDraft.SalesBrief)
			if err != nil {
				return SessionView{}, err
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO assistant_messages (
					tenant_id, conversation_id, direction, sender_type, provider,
					body, status, metadata, created_at
				) VALUES (
					$1, $2, 'OUTBOUND', 'ASSISTANT', 'WEB_CHAT',
					$3, 'DRAFT',
					jsonb_strip_nulls(jsonb_build_object(
						'engine', $4::text,
						'model', $5::text,
						'used_fallback', $6::boolean,
						'fallback_reason', NULLIF($7::text, ''),
						'grounding_references', $8::jsonb,
						'sales_brief', $9::jsonb,
						'human_approval_required', TRUE,
						'source_message_id', $10::text
					)),
					$11
				)
			`,
				tenantID,
				conversationID,
				responseDraft.Body,
				responseDraft.Engine,
				responseDraft.Model,
				responseDraft.UsedFallback,
				string(responseDraft.FallbackReason),
				groundingReferences,
				salesBrief,
				input.ClientMessageID,
				now.Add(time.Microsecond),
			); err != nil {
				return SessionView{}, fmt.Errorf(
					"insert web chat response draft: %w",
					err,
				)
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE assistant_conversations
			SET status = 'HUMAN_REVIEW',
			    last_message_at = $3
			WHERE tenant_id = $1
			  AND id = $2
		`, tenantID, conversationID, now); err != nil {
			return SessionView{}, fmt.Errorf(
				"update web chat conversation: %w",
				err,
			)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE assistant_web_chat_sessions
		SET last_seen_at = $2
		WHERE id = $1
	`, sessionID, now); err != nil {
		return SessionView{}, fmt.Errorf(
			"update web chat session activity: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionView{}, fmt.Errorf(
			"commit web chat message: %w",
			err,
		)
	}

	return r.GetSession(
		ctx,
		tenantSlug,
		sessionID,
		tokenHash,
		now,
	)
}
