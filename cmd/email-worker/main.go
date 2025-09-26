package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"

	"github.com/example/notification-service/internal/common"
	"github.com/example/notification-service/internal/email"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := common.LoadConfig("email-worker")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := common.NewLogger(cfg.ServiceName)
	shutdown, err := common.SetupOTel(ctx, cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialise telemetry")
	}
	defer common.ShutdownTelemetry(context.Background(), shutdown)

	metricsSrv := common.StartMetricsServer(cfg.MetricsPort)
	defer metricsSrv.Shutdown(context.Background())

	readerFactory := func() *kafka.Reader {
		return kafka.NewReader(kafka.ReaderConfig{
			Brokers: cfg.KafkaBrokers,
			GroupID: cfg.ServiceName,
			Topic:   cfg.EmailTopic,
		})
	}

	dlqWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers...),
		Topic:    cfg.DLQTopic,
		Balancer: &kafka.Hash{},
	}
	defer dlqWriter.Close()

	eventWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers...),
		Topic:    cfg.ProviderEventsTopic,
		Balancer: &kafka.Hash{},
	}
	defer eventWriter.Close()

	ses := &email.SESProvider{
		Endpoint: envOr("SES_ENDPOINT", "https://ses.local"),
		APIKey:   os.Getenv("SES_API_KEY"),
	}
	sendgrid := &email.SendGridProvider{
		Endpoint: envOr("SENDGRID_ENDPOINT", "https://sendgrid.local"),
		APIKey:   os.Getenv("SENDGRID_API_KEY"),
	}

	worker := email.Worker{
		ReaderFactory: readerFactory,
		DLQWriter:     dlqWriter,
		EventWriter:   eventWriter,
		Providers:     []email.Provider{ses, sendgrid},
		Logger:        logger,
	}

	logger.Info().Msg("email worker started")
	if err := worker.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("email worker stopped")
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
