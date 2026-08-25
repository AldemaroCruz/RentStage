package webchat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ResolveConfiguration(
	ctx context.Context,
	tenantSlug string,
) (publicConfiguration, error) {
	var item publicConfiguration
	var catalogEnabled bool
	var webChatEnabled bool

	err := r.pool.QueryRow(ctx, `
		SELECT
			tenant.id,
			tenant.name,
			tenant.slug,
			COALESCE(settings.terms_version, ''),
			COALESCE(settings.enabled, FALSE),
			COALESCE(settings.web_chat_enabled, FALSE)
		FROM tenants tenant
		LEFT JOIN public_catalog_settings settings
		  ON settings.tenant_id = tenant.id
		WHERE tenant.slug = $1
		  AND tenant.status = 'ACTIVE'
	`, strings.ToLower(strings.TrimSpace(tenantSlug))).Scan(
		&item.TenantID,
		&item.TenantName,
		&item.TenantSlug,
		&item.TermsVersion,
		&catalogEnabled,
		&webChatEnabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return publicConfiguration{}, ErrNotFound
	}
	if err != nil {
		return publicConfiguration{}, fmt.Errorf(
			"resolve web chat configuration: %w",
			err,
		)
	}
	if !catalogEnabled || !webChatEnabled {
		return publicConfiguration{}, ErrDisabled
	}

	return item, nil
}

func (r *Repository) CreateSession(
	ctx context.Context,
	configuration publicConfiguration,
	input normalizedCreateSession,
	tokenHash string,
	expiresAt time.Time,
	responseDraft string,
	now time.Time,
) (SessionView, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SessionView{}, fmt.Errorf(
			"begin web chat session: %w",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var termsVersion string

	err = tx.QueryRow(ctx, `
		SELECT terms_version
		FROM public_catalog_settings
		WHERE tenant_id = $1
		  AND enabled = TRUE
		  AND web_chat_enabled = TRUE
		FOR SHARE
	`, configuration.TenantID).Scan(&termsVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionView{}, ErrDisabled
	}
	if err != nil {
		return SessionView{}, fmt.Errorf(
			"lock web chat configuration: %w",
			err,
		)
	}

	conversationID := idutil.NewUUID()
	sessionID := idutil.NewUUID()

	if _, err := tx.Exec(ctx, `
		INSERT INTO assistant_conversations (
			id,
			tenant_id,
			channel,
			external_conversation_id,
			contact_name,
			contact_phone,
			contact_email,
			status,
			consent_status,
			summary,
			last_message_at,
			created_at,
			updated_at
		) VALUES (
			$1,
			$2,
			'WEB_CHAT',
			$3,
			$4,
			'',
			$5,
			'HUMAN_REVIEW',
			'OPTED_IN',
			'Conversación iniciada desde el chat web',
			$6,
			$6,
			$6
		)
	`,
		conversationID,
		configuration.TenantID,
		sessionID,
		input.ContactName,
		input.ContactEmail,
		now,
	); err != nil {
		return SessionView{}, fmt.Errorf(
			"insert web chat conversation: %w",
			err,
		)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO assistant_web_chat_sessions (
			id,
			tenant_id,
			conversation_id,
			token_hash,
			status,
			terms_version,
			consent_accepted_at,
			expires_at,
			last_seen_at,
			created_at,
			updated_at
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			'ACTIVE',
			$5,
			$6,
			$7,
			$6,
			$6,
			$6
		)
	`,
		sessionID,
		configuration.TenantID,
		conversationID,
		tokenHash,
		strings.TrimSpace(termsVersion),
		now,
		expiresAt,
	); err != nil {
		return SessionView{}, fmt.Errorf(
			"insert web chat session: %w",
			err,
		)
	}

	if _, err := tx.Exec(ctx, `
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
	`,
		configuration.TenantID,
		conversationID,
		input.ClientMessageID,
		input.Message,
		now,
	); err != nil {
		return SessionView{}, fmt.Errorf(
			"insert initial web chat message: %w",
			err,
		)
	}

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
			configuration.TenantID,
			conversationID,
			responseDraft,
			input.ClientMessageID,
			now.Add(time.Microsecond),
		); err != nil {
			return SessionView{}, fmt.Errorf(
				"insert initial web chat draft: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionView{}, fmt.Errorf(
			"commit web chat session: %w",
			err,
		)
	}

	return r.GetSession(
		ctx,
		configuration.TenantSlug,
		sessionID,
		tokenHash,
		now,
	)
}

func (r *Repository) GetSession(
	ctx context.Context,
	tenantSlug string,
	sessionID string,
	tokenHash string,
	now time.Time,
) (SessionView, error) {
	var item SessionView
	var tenantID string
	var conversationID string

	err := r.pool.QueryRow(ctx, `
		SELECT
			session.id,
			session.status,
			conversation.contact_name,
			session.expires_at,
			session.conversation_id,
			session.tenant_id
		FROM assistant_web_chat_sessions session
		JOIN assistant_conversations conversation
		  ON conversation.tenant_id = session.tenant_id
		 AND conversation.id = session.conversation_id
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
	`,
		strings.ToLower(strings.TrimSpace(tenantSlug)),
		sessionID,
		tokenHash,
	).Scan(
		&item.ID,
		&item.Status,
		&item.ContactName,
		&item.ExpiresAt,
		&conversationID,
		&tenantID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionView{}, ErrInvalidToken
	}
	if err != nil {
		return SessionView{}, fmt.Errorf(
			"get web chat session: %w",
			err,
		)
	}
	if item.Status != "ACTIVE" {
		return SessionView{}, ErrSessionClosed
	}
	if !now.Before(item.ExpiresAt) {
		_, _ = r.pool.Exec(ctx, `
			UPDATE assistant_web_chat_sessions
			SET status = 'CLOSED',
			    updated_at = $2
			WHERE id = $1
			  AND status = 'ACTIVE'
		`, sessionID, now)

		return SessionView{}, ErrSessionExpired
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			direction,
			sender_type,
			body,
			status,
			created_at
		FROM assistant_messages
		WHERE tenant_id = $1
		  AND conversation_id = $2
		  AND (
			direction = 'INBOUND'
			OR (
				direction = 'OUTBOUND'
				AND status IN ('SENT', 'DELIVERED', 'READ')
			)
		  )
		ORDER BY created_at, id
	`, tenantID, conversationID)
	if err != nil {
		return SessionView{}, fmt.Errorf(
			"list public web chat messages: %w",
			err,
		)
	}
	defer rows.Close()

	item.Messages = make([]PublicMessage, 0)

	for rows.Next() {
		var message PublicMessage

		if err := rows.Scan(
			&message.ID,
			&message.Direction,
			&message.SenderType,
			&message.Body,
			&message.Status,
			&message.CreatedAt,
		); err != nil {
			return SessionView{}, fmt.Errorf(
				"scan public web chat message: %w",
				err,
			)
		}

		item.Messages = append(item.Messages, message)
	}
	if err := rows.Err(); err != nil {
		return SessionView{}, fmt.Errorf(
			"iterate public web chat messages: %w",
			err,
		)
	}

	_, _ = r.pool.Exec(ctx, `
		UPDATE assistant_web_chat_sessions
		SET last_seen_at = $2
		WHERE id = $1
	`, sessionID, now)

	return item, nil
}

func generateSessionToken() (string, string, error) {
	value := make([]byte, 32)

	if _, err := rand.Read(value); err != nil {
		return "", "", fmt.Errorf(
			"generate web chat token: %w",
			err,
		)
	}

	rawToken := base64.RawURLEncoding.EncodeToString(value)

	return rawToken, hashSessionToken(rawToken), nil
}

func hashSessionToken(rawToken string) string {
	digest := sha256.Sum256(
		[]byte(strings.TrimSpace(rawToken)),
	)

	return hex.EncodeToString(digest[:])
}
