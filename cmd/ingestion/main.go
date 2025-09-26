package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"

	"github.com/example/notification-service/internal/common"
	"github.com/example/notification-service/internal/ingest"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := common.LoadConfig("ingestion")
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

	if cfg.DatabaseURL == "" {
		logger.Fatal().Msg("DATABASE_URL must be provided")
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect postgres")
	}
	defer pool.Close()

	repo := ingest.NewPostgresRepository(pool)

	producer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers...),
		Topic:    cfg.NotificationTopic,
		Balancer: &kafka.Hash{},
	}
	defer producer.Close()

	h := ingest.NewHandler(repo, producer, cfg, logger)

	srv := &http.Server{
		Addr:    formatAddr(cfg.HTTPPort),
		Handler: h.Router(),
	}

	go func() {
		logger.Info().Int("port", cfg.HTTPPort).Msg("ingestion service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("http server failed")
		}
	}()

	<-ctx.Done()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
	}
}

func formatAddr(port int) string {
	return ":" + strconv.Itoa(port)
}
