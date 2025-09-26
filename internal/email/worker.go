package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Provider interface {
	Name() string
	Send(ctx context.Context, msg Message) error
}

type Message struct {
	MessageID string         `json:"message_id"`
	TenantID  string         `json:"tenant_id"`
	Channel   string         `json:"channel"`
	Payload   map[string]any `json:"payload"`
	Template  string         `json:"template_id"`
	CreatedAt time.Time      `json:"created_at"`
}

type Worker struct {
	ReaderFactory func() *kafka.Reader
	DLQWriter     *kafka.Writer
	EventWriter   *kafka.Writer
	Providers     []Provider
	Logger        zerolog.Logger
}

func (w *Worker) Run(ctx context.Context) error {
	if len(w.Providers) == 0 {
		return errors.New("at least one provider required")
	}
	reader := w.ReaderFactory()
	defer reader.Close()
	tracer := otel.Tracer("email-worker")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetch message: %w", err)
		}

		var payload Message
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			w.Logger.Error().Err(err).Msg("failed to decode email payload")
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		spanCtx, span := tracer.Start(ctx, "deliver_email")
		span.SetAttributes(attribute.String("message.id", payload.MessageID))

		sent := false
		for _, provider := range w.Providers {
			if err := w.deliverWithProvider(spanCtx, provider, payload); err != nil {
				span.RecordError(err)
				w.Logger.Warn().Err(err).Str("provider", provider.Name()).Msg("provider send failed")
				continue
			}
			sent = true
			break
		}

		if !sent {
			w.Logger.Error().Str("message_id", payload.MessageID).Msg("all providers failed, sending to DLQ")
			if err := w.writeDLQ(ctx, payload); err != nil {
				span.RecordError(err)
				span.End()
				return err
			}
		} else {
			if err := w.emitEvent(ctx, payload, "sent"); err != nil {
				span.RecordError(err)
				span.End()
				return err
			}
		}

		span.End()
		if err := reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit message: %w", err)
		}
	}
}

func (w *Worker) deliverWithProvider(ctx context.Context, provider Provider, msg Message) error {
	op := backoff.NewExponentialBackOff()
	op.MaxElapsedTime = 5 * time.Second
	return backoff.Retry(func() error {
		attemptCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := provider.Send(attemptCtx, msg); err != nil {
			return err
		}
		return nil
	}, op)
}

func (w *Worker) writeDLQ(ctx context.Context, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal dlq message: %w", err)
	}
	return w.DLQWriter.WriteMessages(ctx, kafka.Message{Key: []byte(msg.MessageID), Value: payload})
}

func (w *Worker) emitEvent(ctx context.Context, msg Message, status string) error {
	event := map[string]any{
		"message_id":  msg.MessageID,
		"tenant_id":   msg.TenantID,
		"status":      status,
		"channel":     msg.Channel,
		"template_id": msg.Template,
		"emitted_at":  time.Now().UTC(),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return w.EventWriter.WriteMessages(ctx, kafka.Message{Key: []byte(msg.MessageID), Value: payload})
}
