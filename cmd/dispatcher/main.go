package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"

	"github.com/example/notification-service/internal/common"
	"github.com/example/notification-service/internal/dispatcher"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := common.LoadConfig("dispatcher")
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
			Topic:   cfg.NotificationTopic,
		})
	}

	writerCache := map[string]*kafka.Writer{}
	writerFactory := func(topic string) *kafka.Writer {
		if w, ok := writerCache[topic]; ok {
			return w
		}
		writer := &kafka.Writer{
			Addr:     kafka.TCP(cfg.KafkaBrokers...),
			Topic:    topic,
			Balancer: &kafka.Hash{},
		}
		writerCache[topic] = writer
		return writer
	}

	d := dispatcher.Dispatcher{
		ReaderFactory: readerFactory,
		WriterFactory: writerFactory,
		Logger:        logger,
	}

	go func() {
		logger.Info().Msg("dispatcher service started")
		if err := d.Run(ctx); err != nil {
			logger.Fatal().Err(err).Msg("dispatcher stopped")
		}
	}()

	<-ctx.Done()
	for _, writer := range writerCache {
		_ = writer.Close()
	}
}
