package webchat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) AddInboundMessage(
	ctx context.Context,
	tenantSlug string,
	sessionID string,
	tokenHash string,
	input normalizedMessage,
	responseDraft string,
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
		if strings.TrimSpace(responseDraft) != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO assistant_messages (
					tenant_id,
					conversation_id,
					direction,
					sender_type,
					provider,
					body,
					status,
					metadata,
					created_at
				) VALUES (
					$1,
					$2,
					'OUTBOUND',
					'ASSISTANT',
					'WEB_CHAT',
					$3,
					'DRAFT',
					jsonb_build_object(
						'engine', 'WEB_CHAT_RULES',
						'human_approval_required', TRUE,
						'source_message_id', $4::text
					),
					$5
				)
			`,
				tenantID,
				conversationID,
				responseDraft,
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
