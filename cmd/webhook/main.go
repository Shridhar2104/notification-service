package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/example/notification-service/internal/common"
	"github.com/example/notification-service/internal/webhook"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := common.LoadConfig("webhook")
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

	producer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers...),
		Topic:    cfg.ProviderEventsTopic,
		Balancer: &kafka.Hash{},
	}
	defer producer.Close()

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.HTTPPort),
		Handler: webhook.Server{Producer: producer, Logger: logger}.Router(),
	}

	go func() {
		logger.Info().Int("port", cfg.HTTPPort).Msg("webhook service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("webhook server failed")
		}
	}()

	<-ctx.Done()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
	}
}
