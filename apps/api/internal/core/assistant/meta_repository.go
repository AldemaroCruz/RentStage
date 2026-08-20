package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	metaintegration "github.com/rentstage/rentstage/apps/api/internal/integrations/meta"
)

func (r *Repository) ApplyMetaWebhook(
	ctx context.Context,
	events metaintegration.WebhookEvents,
) (metaintegration.ProcessResult, error) {
	result := metaintegration.ProcessResult{}
	for _, inbound := range events.Inbound {
		if inbound.Type != "text" || strings.TrimSpace(inbound.Text) == "" {
			result.Ignored++
			continue
		}
		inserted, ignored, err := r.applyMetaInbound(ctx, inbound)
		if err != nil {
			return result, err
		}
		if ignored {
			result.Ignored++
		} else if inserted {
			result.InboundProcessed++
		} else {
			result.Duplicates++
		}
	}
	for _, status := range events.Statuses {
		processed, err := r.applyMetaStatus(ctx, status)
		if err != nil {
			return result, err
		}
		if processed {
			result.StatusesProcessed++
		} else {
			result.Ignored++
		}
	}
	return result, nil
}

func (r *Repository) applyMetaInbound(
	ctx context.Context,
	message metaintegration.InboundMessage,
) (bool, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, false, fmt.Errorf("begin Meta inbound: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tenantID, err := resolveMetaTenant(ctx, tx, message.PhoneNumberID, message.WABAID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, true, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("resolve Meta tenant: %w", err)
	}
	contactName := strings.TrimSpace(message.ContactName)
	if contactName == "" {
		contactName = "Contacto WhatsApp"
	}
	contactPhone := normalizeWhatsAppPhone(message.From)
	serviceWindow := message.OccurredAt.Add(24 * time.Hour)
	var conversationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO assistant_conversations (
			tenant_id, channel, external_conversation_id, contact_name,
			contact_phone, status, consent_status, service_window_expires_at,
			summary, last_message_at
		) VALUES (
			$1, 'WHATSAPP', $2, $3, $4, 'HUMAN_REVIEW', 'UNKNOWN', $5,
			'Conversación recibida por WhatsApp', $6
		)
		ON CONFLICT (tenant_id, channel, external_conversation_id)
			WHERE external_conversation_id IS NOT NULL
		DO UPDATE SET
			contact_name = EXCLUDED.contact_name,
			contact_phone = EXCLUDED.contact_phone,
			service_window_expires_at = GREATEST(
				assistant_conversations.service_window_expires_at,
				EXCLUDED.service_window_expires_at
			),
			last_message_at = GREATEST(
				assistant_conversations.last_message_at,
				EXCLUDED.last_message_at
			)
		RETURNING id
	`, tenantID, message.From, contactName, contactPhone, serviceWindow, message.OccurredAt).Scan(&conversationID)
	if err != nil {
		return false, false, fmt.Errorf("upsert Meta conversation: %w", err)
	}

	metadata, _ := json.Marshal(map[string]any{
		"transport":            "META_WEBHOOK",
		"waba_id":              message.WABAID,
		"phone_number_id":      message.PhoneNumberID,
		"display_phone_number": message.DisplayPhone,
		"message_type":         message.Type,
	})
	var insertedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO assistant_messages (
			tenant_id, conversation_id, direction, sender_type, provider,
			external_message_id, body, status, metadata, created_at
		) VALUES ($1, $2, 'INBOUND', 'CUSTOMER', 'WHATSAPP', $3, $4, 'RECEIVED', $5::jsonb, $6)
		ON CONFLICT (tenant_id, provider, external_message_id)
			WHERE external_message_id IS NOT NULL
		DO NOTHING
		RETURNING id
	`, tenantID, conversationID, message.MessageID, message.Text, string(metadata), message.OccurredAt).Scan(&insertedID)
	if errors.Is(err, pgx.ErrNoRows) {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return false, false, fmt.Errorf("commit duplicate Meta inbound: %w", commitErr)
		}
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("insert Meta inbound: %w", err)
	}

	draftMetadata, _ := json.Marshal(map[string]any{
		"engine":                  "META_LOCAL_RULES",
		"human_approval_required": true,
		"source_message_id":       message.MessageID,
	})
	draft := fmt.Sprintf(
		"¡Hola, %s! Gracias por escribirnos. Recibimos tu mensaje y un miembro del equipo preparará la información para responderte.",
		contactName,
	)
	if _, err := tx.Exec(ctx, `
		INSERT INTO assistant_messages (
			tenant_id, conversation_id, direction, sender_type, provider,
			body, status, metadata
		) VALUES ($1, $2, 'OUTBOUND', 'ASSISTANT', 'WHATSAPP', $3, 'DRAFT', $4::jsonb)
	`, tenantID, conversationID, draft, string(draftMetadata)); err != nil {
		return false, false, fmt.Errorf("insert Meta response draft: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE assistant_conversations
		SET status = 'HUMAN_REVIEW', last_message_at = $3
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, conversationID, message.OccurredAt); err != nil {
		return false, false, fmt.Errorf("update Meta conversation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, false, fmt.Errorf("commit Meta inbound: %w", err)
	}
	return true, false, nil
}

func (r *Repository) applyMetaStatus(
	ctx context.Context,
	status metaintegration.StatusUpdate,
) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin Meta status: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveMetaTenant(ctx, tx, status.PhoneNumberID, status.WABAID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("resolve Meta status tenant: %w", err)
	}
	metadata, _ := json.Marshal(map[string]any{
		"delivery_status":     status.Status,
		"delivery_updated_at": status.OccurredAt,
		"delivery_errors":     status.Errors,
	})
	messageStatus := "SENT"
	if status.Status == "failed" {
		messageStatus = "FAILED"
	}
	command, err := tx.Exec(ctx, `
		UPDATE assistant_messages
		SET status = $3, metadata = metadata || $4::jsonb
		WHERE tenant_id = $1 AND provider = 'WHATSAPP'
		  AND external_message_id = $2 AND direction = 'OUTBOUND'
	`, tenantID, status.MessageID, messageStatus, string(metadata))
	if err != nil {
		return false, fmt.Errorf("update Meta status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit Meta status: %w", err)
	}
	return command.RowsAffected() > 0, nil
}

func resolveMetaTenant(ctx context.Context, tx pgx.Tx, phoneNumberID, wabaID string) (string, error) {
	var tenantID string
	err := tx.QueryRow(ctx, `
		SELECT tenant_id
		FROM assistant_channel_connections
		WHERE provider = 'WHATSAPP'
		  AND phone_number_id = $1
		  AND waba_id = $2
		  AND enabled
	`, strings.TrimSpace(phoneNumberID), strings.TrimSpace(wabaID)).Scan(&tenantID)
	return tenantID, err
}

func normalizeWhatsAppPhone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "+") {
		return value
	}
	return "+" + value
}

func (r *Repository) RecordWhatsAppSent(
	ctx context.Context,
	tenantID, conversationID, messageID, externalMessageID, body, actorID string,
) (ConversationDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("begin WhatsApp message delivery: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var channel string
	if err := tx.QueryRow(ctx, `
		SELECT channel FROM assistant_conversations
		WHERE tenant_id = $1 AND id = $2 FOR UPDATE
	`, tenantID, conversationID).Scan(&channel); errors.Is(err, pgx.ErrNoRows) {
		return ConversationDetail{}, ErrNotFound
	} else if err != nil {
		return ConversationDetail{}, fmt.Errorf("lock WhatsApp conversation: %w", err)
	}
	if channel != "WHATSAPP" {
		return ConversationDetail{}, ErrDemoOnly
	}
	metadata, _ := json.Marshal(map[string]any{
		"human_approved": true,
		"transport":      "META_GRAPH_API",
	})
	if messageID == "" {
		_, err = tx.Exec(ctx, `
			INSERT INTO assistant_messages (
				tenant_id, conversation_id, direction, sender_type, provider,
				external_message_id, body, status, metadata, created_by,
				approved_by, approved_at
			) VALUES (
				$1, $2, 'OUTBOUND', 'USER', 'WHATSAPP', $3, $4, 'SENT', $5::jsonb,
				NULLIF($6, '')::uuid, NULLIF($6, '')::uuid, NOW()
			)
		`, tenantID, conversationID, externalMessageID, body, string(metadata), actorID)
	} else {
		result, updateErr := tx.Exec(ctx, `
			UPDATE assistant_messages
			SET body = $4, status = 'SENT', external_message_id = $5,
				approved_by = NULLIF($6, '')::uuid, approved_at = NOW(),
				metadata = metadata || $7::jsonb
			WHERE tenant_id = $1 AND conversation_id = $2 AND id = $3
			  AND direction = 'OUTBOUND' AND status IN ('DRAFT', 'APPROVED')
		`, tenantID, conversationID, messageID, body, externalMessageID, actorID, string(metadata))
		err = updateErr
		if err == nil && result.RowsAffected() == 0 {
			return ConversationDetail{}, ErrMessageNotReady
		}
	}
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("record WhatsApp delivery: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE assistant_conversations
		SET status = CASE WHEN status = 'HUMAN_REVIEW' THEN 'OPEN' ELSE status END,
			last_message_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, conversationID); err != nil {
		return ConversationDetail{}, fmt.Errorf("update WhatsApp delivery status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ConversationDetail{}, fmt.Errorf("commit WhatsApp delivery: %w", err)
	}
	return r.Get(ctx, tenantID, conversationID)
}
