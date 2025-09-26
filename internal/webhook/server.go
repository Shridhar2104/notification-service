package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/example/notification-service/internal/common"
)

type Server struct {
	Producer *kafka.Writer
	Logger   zerolog.Logger
}

var (
	eventCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "webhook_events_total",
		Help: "Total webhook events processed",
	}, []string{"provider", "status"})
)

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Post("/v1/providers/{provider}/events", s.handle)
	return r
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("webhook").Start(r.Context(), "ingest-webhook")
	defer span.End()

	provider := chi.URLParam(r, "provider")
	if provider == "" {
		s.respondErr(ctx, w, http.StatusBadRequest, errors.New("provider path param required"))
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.respondErr(ctx, w, http.StatusBadRequest, err)
		return
	}

	event, err := s.normalize(provider, payload)
	if err != nil {
		s.respondErr(ctx, w, http.StatusBadRequest, err)
		return
	}
	span.SetAttributes(attribute.String("message.id", event.MessageID))

	body, err := json.Marshal(event)
	if err != nil {
		s.respondErr(ctx, w, http.StatusInternalServerError, err)
		return
	}

	if err := s.Producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.MessageID),
		Value: body,
	}); err != nil {
		s.respondErr(ctx, w, http.StatusInternalServerError, err)
		return
	}

	eventCounter.WithLabelValues(provider, "ok").Inc()
	w.WriteHeader(http.StatusAccepted)
}

type NormalizedEvent struct {
	MessageID string                 `json:"message_id"`
	TenantID  string                 `json:"tenant_id"`
	Provider  string                 `json:"provider"`
	Status    string                 `json:"status"`
	Occurred  time.Time              `json:"occurred_at"`
	Meta      map[string]interface{} `json:"meta"`
}

func (s *Server) normalize(provider string, payload map[string]any) (NormalizedEvent, error) {
	switch provider {
	case "ses":
		return s.normalizeSES(payload)
	case "sendgrid":
		return s.normalizeSendGrid(payload)
	default:
		return NormalizedEvent{}, errors.New("unsupported provider")
	}
}

func (s *Server) normalizeSES(payload map[string]any) (NormalizedEvent, error) {
	messageID, _ := payload["message_id"].(string)
	if messageID == "" {
		return NormalizedEvent{}, errors.New("ses message_id missing")
	}
	status, _ := payload["event"].(string)
	if status == "" {
		return NormalizedEvent{}, errors.New("ses event missing")
	}
	tenant, _ := payload["tenant_id"].(string)
	return NormalizedEvent{
		MessageID: messageID,
		TenantID:  tenant,
		Provider:  "ses",
		Status:    status,
		Occurred:  time.Now().UTC(),
		Meta:      payload,
	}, nil
}

func (s *Server) normalizeSendGrid(payload map[string]any) (NormalizedEvent, error) {
	messageID, _ := payload["sg_message_id"].(string)
	if messageID == "" {
		return NormalizedEvent{}, errors.New("sendgrid sg_message_id missing")
	}
	status, _ := payload["event"].(string)
	if status == "" {
		return NormalizedEvent{}, errors.New("sendgrid event missing")
	}
	tenant, _ := payload["tenant_id"].(string)
	return NormalizedEvent{
		MessageID: messageID,
		TenantID:  tenant,
		Provider:  "sendgrid",
		Status:    status,
		Occurred:  time.Now().UTC(),
		Meta:      payload,
	}, nil
}

func (s *Server) respondErr(ctx context.Context, w http.ResponseWriter, status int, err error) {
	logger := common.WithContext(ctx, s.Logger)
	logger.Error().Err(err).Int("status", status).Msg("webhook handler error")
	eventCounter.WithLabelValues("unknown", "error").Inc()
	http.Error(w, err.Error(), status)
}
