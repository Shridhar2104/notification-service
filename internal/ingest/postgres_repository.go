package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const insertMessage = `
INSERT INTO messages (
id,
tenant_id,
message_key,
channel,
payload_json,
template_id,
status,
created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (tenant_id, message_key) DO NOTHING
RETURNING id, tenant_id, message_key, channel, payload_json, template_id, status, created_at
`

const selectMessage = `
SELECT id, tenant_id, message_key, channel, payload_json, template_id, status, created_at
FROM messages
WHERE tenant_id = $1 AND message_key = $2
`

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateMessage(ctx context.Context, msg Message) (Message, bool, error) {
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return Message{}, false, err
	}

	row := r.pool.QueryRow(ctx, insertMessage,
		msg.ID,
		msg.TenantID,
		msg.MessageKey,
		string(msg.Channel),
		payload,
		msg.TemplateID,
		msg.Status,
		msg.CreatedAt,
	)

	var (
		id          string
		tenantID    string
		messageKey  string
		channel     string
		payloadJSON []byte
		templateID  string
		status      string
		createdAt   time.Time
	)

	inserted := true
	if err := row.Scan(&id, &tenantID, &messageKey, &channel, &payloadJSON, &templateID, &status, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			inserted = false
			row = r.pool.QueryRow(ctx, selectMessage, msg.TenantID, msg.MessageKey)
			if err := row.Scan(&id, &tenantID, &messageKey, &channel, &payloadJSON, &templateID, &status, &createdAt); err != nil {
				return Message{}, false, fmt.Errorf("fetch existing message: %w", err)
			}
		} else {
			return Message{}, false, fmt.Errorf("insert message: %w", err)
		}
	}

	var payloadMap map[string]any
	if err := json.Unmarshal(payloadJSON, &payloadMap); err != nil {
		return Message{}, false, err
	}

	return Message{
		ID:         id,
		TenantID:   tenantID,
		MessageKey: messageKey,
		Channel:    Channel(channel),
		Payload:    payloadMap,
		TemplateID: templateID,
		Status:     status,
		CreatedAt:  createdAt,
	}, !inserted, nil
}

var ErrNotConfigured = errors.New("postgres repository requires a non-nil pool")

func MustRepository(pool *pgxpool.Pool) (*PostgresRepository, error) {
	if pool == nil {
		return nil, ErrNotConfigured
	}
	return NewPostgresRepository(pool), nil
}
