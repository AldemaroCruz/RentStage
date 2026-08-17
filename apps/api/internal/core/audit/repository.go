package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Repository struct {
	pool *pgxpool.Pool
}

type Event struct {
	ID         string         `json:"id"`
	ActorType  string         `json:"actor_type"`
	ActorID    string         `json:"actor_id"`
	ActorName  *string        `json:"actor_name,omitempty"`
	ActorEmail *string        `json:"actor_email,omitempty"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   *string        `json:"entity_id,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	RequestID  *string        `json:"request_id,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Record(
	ctx context.Context,
	tenantID string,
	action string,
	entityType string,
	entityID *string,
	metadata map[string]any,
) error {
	return r.RecordAs(ctx, tenantID, "USER", webutil.ActorID(ctx), action, entityType, entityID, metadata)
}

func (r *Repository) RecordAs(
	ctx context.Context,
	tenantID string,
	actorType string,
	actorID string,
	action string,
	entityType string,
	entityID *string,
	metadata map[string]any,
) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO audit_events (
			tenant_id, actor_type, actor_id, action, entity_type, entity_id, metadata, request_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, NULLIF($8, ''))
	`, tenantID, actorType, actorID, action, entityType, entityID, string(payload), webutil.RequestID(ctx))
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, tenantID string, limit int) ([]Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ae.id, ae.actor_type, ae.actor_id, u.display_name, u.email,
		       ae.action, ae.entity_type, ae.entity_id,
		       ae.metadata, ae.request_id, ae.created_at
		FROM audit_events ae
		LEFT JOIN users u ON u.id::text = ae.actor_id
		WHERE ae.tenant_id = $1
		ORDER BY ae.created_at DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		var metadataBytes []byte
		if err := rows.Scan(
			&event.ID,
			&event.ActorType,
			&event.ActorID,
			&event.ActorName,
			&event.ActorEmail,
			&event.Action,
			&event.EntityType,
			&event.EntityID,
			&metadataBytes,
			&event.RequestID,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		if err := json.Unmarshal(metadataBytes, &event.Metadata); err != nil {
			event.Metadata = map[string]any{}
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
