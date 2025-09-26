package ingest

import (
	"context"
	"time"
)

type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelPush     Channel = "push"
	ChannelWhatsApp Channel = "whatsapp"
)

type NotifyRequest struct {
	Channel    Channel        `json:"channel"`
	To         map[string]any `json:"to"`
	TemplateID string         `json:"template_id"`
	Data       map[string]any `json:"data"`
	Options    map[string]any `json:"options"`
}

type Message struct {
	ID         string
	TenantID   string
	MessageKey string
	Channel    Channel
	Payload    map[string]any
	TemplateID string
	Status     string
	CreatedAt  time.Time
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, msg Message) (Message, bool, error)
}
