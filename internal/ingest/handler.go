package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/example/notification-service/internal/common"
)

var (
	reqCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingest_requests_total",
		Help: "Total number of /notify requests received",
	}, []string{"status", "channel"})
	requestLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ingest_request_duration_seconds",
		Help:    "Latency for /notify requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"channel"})
)

type Handler struct {
	repo     MessageRepository
	producer *kafka.Writer
	cfg      *common.Config
	tracer   trace.Tracer
	logger   zerolog.Logger
}

func NewHandler(repo MessageRepository, producer *kafka.Writer, cfg *common.Config, logger zerolog.Logger) *Handler {
	return &Handler{
		repo:     repo,
		producer: producer,
		cfg:      cfg,
		tracer:   otel.Tracer("ingestion"),
		logger:   logger,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Post("/v1/notify", h.notify)
	return r
}

func (h *Handler) notify(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "notify")
	defer span.End()

	tenantID := r.Header.Get("x-tenant-id")
	if tenantID == "" {
		h.respondErr(ctx, w, http.StatusBadRequest, errors.New("missing x-tenant-id header"))
		return
	}
	idempotencyKey := r.Header.Get("x-idempotency-key")
	if idempotencyKey == "" {
		h.respondErr(ctx, w, http.StatusBadRequest, errors.New("missing x-idempotency-key header"))
		return
	}

	var req NotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondErr(ctx, w, http.StatusBadRequest, err)
		return
	}
	if err := validateRequest(req); err != nil {
		h.respondErr(ctx, w, http.StatusBadRequest, err)
		return
	}

	start := time.Now()

	msg := Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		MessageKey: idempotencyKey,
		Channel:    req.Channel,
		TemplateID: req.TemplateID,
		Payload: map[string]any{
			"to":      req.To,
			"data":    req.Data,
			"options": req.Options,
		},
		Status:    "queued",
		CreatedAt: time.Now().UTC(),
	}

	saved, duplicate, err := h.repo.CreateMessage(ctx, msg)
	if err != nil {
		h.respondErr(ctx, w, http.StatusInternalServerError, err)
		return
	}
	msg = saved

	reqCounter.WithLabelValues(statusLabel(duplicate), string(req.Channel)).Inc()
	requestLatency.WithLabelValues(string(req.Channel)).Observe(time.Since(start).Seconds())

	if duplicate {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message_id": msg.ID,
			"status":     "duplicate",
		})
		return
	}

	event := map[string]any{
		"message_id":  msg.ID,
		"tenant_id":   msg.TenantID,
		"channel":     msg.Channel,
		"payload":     msg.Payload,
		"template_id": msg.TemplateID,
		"created_at":  msg.CreatedAt,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		h.respondErr(ctx, w, http.StatusInternalServerError, err)
		return
	}
	span.SetAttributes(attribute.String("message.id", msg.ID))

	if err := h.producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.TenantID + ":" + msg.MessageKey),
		Value: payload,
	}); err != nil {
		h.respondErr(ctx, w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"message_id": msg.ID})
}

func (h *Handler) respondErr(ctx context.Context, w http.ResponseWriter, status int, err error) {
	logger := common.WithContext(ctx, h.logger)
	logger.Error().Err(err).Int("status", status).Msg("notify handler failed")
	reqCounter.WithLabelValues(http.StatusText(status), "unknown").Inc()
	http.Error(w, err.Error(), status)
}

func statusLabel(duplicate bool) string {
	if duplicate {
		return "duplicate"
	}
	return "accepted"
}

func validateRequest(req NotifyRequest) error {
	if req.Channel == "" {
		return errors.New("channel is required")
	}
	if req.TemplateID == "" {
		return errors.New("template_id is required")
	}
	if len(req.To) == 0 {
		return errors.New("to is required")
	}
	return nil
}
