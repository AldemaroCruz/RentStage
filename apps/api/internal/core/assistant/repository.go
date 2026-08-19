package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const conversationSummarySelect = `
	SELECT
		conversation.id,
		conversation.channel,
		conversation.customer_id,
		CASE WHEN customer.id IS NULL THEN NULL ELSE TRIM(customer.first_name || ' ' || customer.last_name) END,
		conversation.contact_name,
		conversation.contact_phone,
		conversation.status,
		conversation.consent_status,
		conversation.summary,
		COALESCE(last_message.body, ''),
		conversation.last_message_at,
		proposal.quote_id,
		quote.quote_number,
		conversation.created_at,
		conversation.updated_at
	FROM assistant_conversations conversation
	LEFT JOIN customers customer
	  ON customer.tenant_id = conversation.tenant_id AND customer.id = conversation.customer_id
	LEFT JOIN assistant_proposals proposal
	  ON proposal.tenant_id = conversation.tenant_id AND proposal.conversation_id = conversation.id
	LEFT JOIN quotes quote
	  ON quote.tenant_id = proposal.tenant_id AND quote.id = proposal.quote_id
	LEFT JOIN LATERAL (
		SELECT message.body
		FROM assistant_messages message
		WHERE message.tenant_id = conversation.tenant_id
		  AND message.conversation_id = conversation.id
		ORDER BY message.created_at DESC, message.id DESC
		LIMIT 1
	) last_message ON TRUE
`

func (r *Repository) List(ctx context.Context, tenantID string) ([]ConversationSummary, error) {
	rows, err := r.pool.Query(ctx, conversationSummarySelect+`
		WHERE conversation.tenant_id = $1
		ORDER BY conversation.last_message_at DESC, conversation.created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list assistant conversations: %w", err)
	}
	defer rows.Close()

	items := make([]ConversationSummary, 0)
	for rows.Next() {
		item, scanErr := scanConversationSummary(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Get(ctx context.Context, tenantID, conversationID string) (ConversationDetail, error) {
	summary, err := scanConversationSummary(r.pool.QueryRow(ctx, conversationSummarySelect+`
		WHERE conversation.tenant_id = $1 AND conversation.id = $2
	`, tenantID, conversationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ConversationDetail{}, ErrNotFound
	}
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("get assistant conversation: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, direction, sender_type, provider, body, status, metadata,
		       approved_by, approved_at, created_at
		FROM assistant_messages
		WHERE tenant_id = $1 AND conversation_id = $2
		ORDER BY created_at, id
	`, tenantID, conversationID)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("list assistant messages: %w", err)
	}
	defer rows.Close()
	messages := make([]Message, 0)
	for rows.Next() {
		var item Message
		var metadata []byte
		if err := rows.Scan(
			&item.ID, &item.Direction, &item.SenderType, &item.Provider,
			&item.Body, &item.Status, &metadata, &item.ApprovedBy,
			&item.ApprovedAt, &item.CreatedAt,
		); err != nil {
			return ConversationDetail{}, fmt.Errorf("scan assistant message: %w", err)
		}
		item.Metadata = decodeMetadata(metadata)
		messages = append(messages, item)
	}
	if err := rows.Err(); err != nil {
		return ConversationDetail{}, fmt.Errorf("iterate assistant messages: %w", err)
	}

	proposal, err := r.getProposal(ctx, tenantID, conversationID)
	if err != nil {
		return ConversationDetail{}, err
	}
	return ConversationDetail{ConversationSummary: summary, Messages: messages, Proposal: proposal}, nil
}

func (r *Repository) CreateDemo(
	ctx context.Context,
	tenantID string,
	input normalizedSimulation,
	proposal proposalRecord,
) (ConversationDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("begin assistant simulation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var conversationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO assistant_conversations (
			tenant_id, channel, contact_name, contact_phone, status,
			consent_status, summary, last_message_at
		) VALUES ($1, 'DEMO', $2, $3, 'HUMAN_REVIEW', 'DEMO', $4, NOW())
		RETURNING id
	`, tenantID, input.ContactName, input.ContactPhone,
		fmt.Sprintf("%s para %d personas en %s", input.EventType, input.GuestCount, input.EventLocation),
	).Scan(&conversationID)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("insert assistant conversation: %w", err)
	}

	intentMetadata, _ := json.Marshal(map[string]any{
		"event_type":     input.EventType,
		"start_at":       input.StartAt,
		"end_at":         input.EndAt,
		"event_location": input.EventLocation,
		"guest_count":    input.GuestCount,
	})
	if _, err := tx.Exec(ctx, `
		INSERT INTO assistant_messages (
			tenant_id, conversation_id, direction, sender_type, provider, body, status, metadata
		) VALUES ($1, $2, 'INBOUND', 'CUSTOMER', 'DEMO', $3, 'RECEIVED', $4::jsonb)
	`, tenantID, conversationID, input.Message, string(intentMetadata)); err != nil {
		return ConversationDetail{}, fmt.Errorf("insert assistant inbound message: %w", err)
	}

	evidence, _ := json.Marshal(proposal.Evidence)
	if _, err := tx.Exec(ctx, `
		INSERT INTO assistant_proposals (
			tenant_id, conversation_id, status, provider, event_type,
			start_at, end_at, event_location, guest_count, package_id,
			package_quantity, package_name, package_price, available,
			recommendation, response_draft, evidence
		) VALUES (
			$1, $2, 'PROPOSED', 'DEMO_RULES', $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15::jsonb
		)
	`, tenantID, conversationID, proposal.EventType, proposal.StartAt, proposal.EndAt,
		proposal.EventLocation, proposal.GuestCount, proposal.PackageID,
		proposal.PackageQuantity, proposal.PackageName, proposal.PackagePrice,
		proposal.Available, proposal.Recommendation, proposal.ResponseDraft,
		string(evidence)); err != nil {
		return ConversationDetail{}, fmt.Errorf("insert assistant proposal: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO assistant_messages (
			tenant_id, conversation_id, direction, sender_type, provider, body, status,
			metadata
		) VALUES ($1, $2, 'OUTBOUND', 'ASSISTANT', 'DEMO', $3, 'DRAFT', $4::jsonb)
	`, tenantID, conversationID, proposal.ResponseDraft, string(evidence)); err != nil {
		return ConversationDetail{}, fmt.Errorf("insert assistant draft: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ConversationDetail{}, fmt.Errorf("commit assistant simulation: %w", err)
	}
	return r.Get(ctx, tenantID, conversationID)
}

func (r *Repository) Approve(
	ctx context.Context,
	tenantID, conversationID, customerID, quoteID, responseBody, actorID string,
) (ConversationDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("begin assistant approval: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	command, err := tx.Exec(ctx, `
		UPDATE assistant_proposals
		SET status = 'QUOTE_CREATED', quote_id = $3, response_draft = $4,
		    approved_by = NULLIF($5, '')::uuid, approved_at = NOW()
		WHERE tenant_id = $1 AND conversation_id = $2
		  AND status = 'PROPOSED' AND quote_id IS NULL
	`, tenantID, conversationID, quoteID, responseBody, actorID)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("approve assistant proposal: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ConversationDetail{}, ErrAlreadyApproved
	}

	if _, err := tx.Exec(ctx, `
		UPDATE assistant_conversations
		SET customer_id = $3, status = 'QUOTE_DRAFTED', last_message_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, conversationID, customerID); err != nil {
		return ConversationDetail{}, fmt.Errorf("update assistant conversation: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE assistant_messages
		SET status = 'APPROVED', body = $3, approved_by = NULLIF($4, '')::uuid, approved_at = NOW()
		WHERE tenant_id = $1 AND conversation_id = $2
		  AND sender_type = 'ASSISTANT' AND status = 'DRAFT'
	`, tenantID, conversationID, responseBody, actorID); err != nil {
		return ConversationDetail{}, fmt.Errorf("approve assistant response: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ConversationDetail{}, fmt.Errorf("commit assistant approval: %w", err)
	}
	return r.Get(ctx, tenantID, conversationID)
}

func (r *Repository) getProposal(ctx context.Context, tenantID, conversationID string) (*Proposal, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT proposal.id, proposal.status, proposal.provider, proposal.event_type,
		       proposal.start_at, proposal.end_at, proposal.event_location,
		       proposal.guest_count, proposal.package_id, proposal.package_quantity,
		       proposal.package_name, proposal.package_price::float8,
		       proposal.available, proposal.recommendation, proposal.response_draft,
		       proposal.evidence, proposal.quote_id, quote.quote_number,
		       proposal.approved_by, proposal.approved_at,
		       proposal.created_at, proposal.updated_at
		FROM assistant_proposals proposal
		LEFT JOIN quotes quote
		  ON quote.tenant_id = proposal.tenant_id AND quote.id = proposal.quote_id
		WHERE proposal.tenant_id = $1 AND proposal.conversation_id = $2
	`, tenantID, conversationID)
	var item Proposal
	var evidence []byte
	if err := row.Scan(
		&item.ID, &item.Status, &item.Provider, &item.EventType,
		&item.StartAt, &item.EndAt, &item.EventLocation, &item.GuestCount,
		&item.PackageID, &item.PackageQuantity, &item.PackageName,
		&item.PackagePrice, &item.Available, &item.Recommendation,
		&item.ResponseDraft, &evidence, &item.QuoteID, &item.QuoteNumber,
		&item.ApprovedBy, &item.ApprovedAt, &item.CreatedAt, &item.UpdatedAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get assistant proposal: %w", err)
	}
	item.Evidence = decodeMetadata(evidence)
	return &item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanConversationSummary(row rowScanner) (ConversationSummary, error) {
	var item ConversationSummary
	if err := row.Scan(
		&item.ID, &item.Channel, &item.CustomerID, &item.CustomerName,
		&item.ContactName, &item.ContactPhone, &item.Status,
		&item.ConsentStatus, &item.Summary, &item.LastMessage,
		&item.LastMessageAt, &item.QuoteID, &item.QuoteNumber,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return ConversationSummary{}, err
	}
	return item, nil
}

func decodeMetadata(value []byte) map[string]any {
	result := map[string]any{}
	if len(value) > 0 {
		_ = json.Unmarshal(value, &result)
	}
	return result
}
