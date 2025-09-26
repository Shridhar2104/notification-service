package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Dispatcher struct {
	ReaderFactory func() *kafka.Reader
	WriterFactory func(topic string) *kafka.Writer
	Logger        zerolog.Logger
}

type IncomingMessage struct {
	MessageID string                 `json:"message_id"`
	TenantID  string                 `json:"tenant_id"`
	Channel   string                 `json:"channel"`
	Payload   map[string]any         `json:"payload"`
	Template  string                 `json:"template_id"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (d *Dispatcher) Run(ctx context.Context) error {
	if d.ReaderFactory == nil || d.WriterFactory == nil {
		return errors.New("dispatcher requires reader and writer factories")
	}
	reader := d.ReaderFactory()
	defer reader.Close()

	tracer := otel.Tracer("dispatcher")

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetch message: %w", err)
		}
		var incoming IncomingMessage
		if err := json.Unmarshal(m.Value, &incoming); err != nil {
			d.Logger.Error().Err(err).Msg("failed to decode message")
			_ = reader.CommitMessages(ctx, m)
			continue
		}

		spanCtx, span := tracer.Start(ctx, "dispatch")
		span.SetAttributes(attribute.String("message.id", incoming.MessageID))

		topic := topicForChannel(incoming.Channel)
		if topic == "" {
			d.Logger.Warn().Str("channel", incoming.Channel).Msg("unknown channel, sending to DLQ")
			topic = "dlq.notifications"
		}

		writer := d.WriterFactory(topic)
		payload, err := json.Marshal(incoming)
		if err != nil {
			span.RecordError(err)
			span.End()
			d.Logger.Error().Err(err).Msg("failed to marshal payload")
			_ = reader.CommitMessages(ctx, m)
			continue
		}
		if err := writer.WriteMessages(spanCtx, kafka.Message{
			Key:   []byte(incoming.TenantID + ":" + incoming.MessageID),
			Value: payload,
		}); err != nil {
			span.RecordError(err)
			span.End()
			return fmt.Errorf("write message: %w", err)
		}

		span.End()
		if err := reader.CommitMessages(ctx, m); err != nil {
			return fmt.Errorf("commit message: %w", err)
		}
	}
}

func topicForChannel(channel string) string {
	switch channel {
	case "email":
		return "dispatch.email"
	case "sms":
		return "dispatch.sms"
	case "push":
		return "dispatch.push"
	case "whatsapp":
		return "dispatch.wa"
	default:
		return ""
	}
}
