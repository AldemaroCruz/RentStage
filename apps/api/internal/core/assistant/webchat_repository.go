package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) RecordWebChatSent(
	ctx context.Context,
	tenantID, conversationID, messageID, body, actorID string,
) (ConversationDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf(
			"begin web chat message delivery: %w",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var channel string
	err = tx.QueryRow(ctx, `
		SELECT channel
		FROM assistant_conversations
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, conversationID).Scan(&channel)

	if errors.Is(err, pgx.ErrNoRows) {
		return ConversationDetail{}, ErrNotFound
	}
	if err != nil {
		return ConversationDetail{}, fmt.Errorf(
			"lock web chat conversation: %w",
			err,
		)
	}
	if channel != "WEB_CHAT" {
		return ConversationDetail{}, fmt.Errorf(
			"web chat delivery requires WEB_CHAT channel",
		)
	}

	metadata, _ := json.Marshal(map[string]any{
		"human_approved": true,
		"transport":      "PUBLIC_WEB_CHAT",
	})

	sentMessageID := messageID

	if messageID == "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO assistant_messages (
				tenant_id,
				conversation_id,
				direction,
				sender_type,
				provider,
				body,
				status,
				metadata,
				created_by,
				approved_by,
				approved_at
			) VALUES (
				$1,
				$2,
				'OUTBOUND',
				'USER',
				'WEB_CHAT',
				$3,
				'SENT',
				$4::jsonb,
				NULLIF($5, '')::uuid,
				NULLIF($5, '')::uuid,
				NOW()
			)
			RETURNING id
		`, tenantID, conversationID, body, string(metadata), actorID).
			Scan(&sentMessageID)
	} else {
		command, updateErr := tx.Exec(ctx, `
			UPDATE assistant_messages
			SET
				sender_type = 'USER',
				body = $4,
				status = 'SENT',
				created_by = NULLIF($5, '')::uuid,
				approved_by = NULLIF($5, '')::uuid,
				approved_at = NOW(),
				created_at = NOW(),
				metadata = metadata || $6::jsonb
			WHERE tenant_id = $1
			  AND conversation_id = $2
			  AND id = $3
			  AND provider = 'WEB_CHAT'
			  AND direction = 'OUTBOUND'
			  AND status IN ('DRAFT', 'APPROVED')
		`, tenantID, conversationID, messageID, body, actorID, string(metadata))

		err = updateErr
		if err == nil && command.RowsAffected() == 0 {
			return ConversationDetail{}, ErrMessageNotReady
		}
	}

	if err != nil {
		return ConversationDetail{}, fmt.Errorf(
			"record web chat delivery: %w",
			err,
		)
	}

	superseded, _ := json.Marshal(map[string]any{
		"superseded":    true,
		"superseded_by": sentMessageID,
	})

	if _, err := tx.Exec(ctx, `
		UPDATE assistant_messages
		SET
			status = 'FAILED',
			metadata = metadata || $4::jsonb
		WHERE tenant_id = $1
		  AND conversation_id = $2
		  AND id <> $3::uuid
		  AND provider = 'WEB_CHAT'
		  AND direction = 'OUTBOUND'
		  AND status IN ('DRAFT', 'APPROVED')
	`, tenantID, conversationID, sentMessageID, string(superseded)); err != nil {
		return ConversationDetail{}, fmt.Errorf(
			"supersede stale web chat drafts: %w",
			err,
		)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE assistant_conversations conversation
		SET
			status = CASE
				WHEN EXISTS (
					SELECT 1
					FROM assistant_proposals proposal
					WHERE proposal.tenant_id = conversation.tenant_id
					  AND proposal.conversation_id = conversation.id
					  AND proposal.quote_id IS NOT NULL
				)
				THEN 'QUOTE_DRAFTED'
				ELSE 'OPEN'
			END,
			last_message_at = NOW()
		WHERE conversation.tenant_id = $1
		  AND conversation.id = $2
	`, tenantID, conversationID); err != nil {
		return ConversationDetail{}, fmt.Errorf(
			"update web chat delivery status: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return ConversationDetail{}, fmt.Errorf(
			"commit web chat delivery: %w",
			err,
		)
	}

	return r.Get(ctx, tenantID, conversationID)
}
