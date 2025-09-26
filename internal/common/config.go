package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort            int
	MetricsPort         int
	DatabaseURL         string
	KafkaBrokers        []string
	NotificationTopic   string
	EmailTopic          string
	DLQTopic            string
	ProviderEventsTopic string
	OTLPEndpoint        string
	ServiceName         string
}

func LoadConfig(service string) (*Config, error) {
	cfg := &Config{ServiceName: service}

	httpPort, err := getEnvInt("HTTP_PORT", 8080)
	if err != nil {
		return nil, err
	}
	cfg.HTTPPort = httpPort

	metricsPort, err := getEnvInt("METRICS_PORT", httpPort+1000)
	if err != nil {
		return nil, err
	}
	cfg.MetricsPort = metricsPort

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	cfg.OTLPEndpoint = os.Getenv("OTLP_ENDPOINT")

	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		cfg.KafkaBrokers = []string{"localhost:9092"}
	} else {
		cfg.KafkaBrokers = strings.Split(brokers, ",")
	}

	cfg.NotificationTopic = getEnv("NOTIFICATION_TOPIC", "notifications")
	cfg.EmailTopic = getEnv("EMAIL_TOPIC", "dispatch.email")
	cfg.DLQTopic = getEnv("DLQ_TOPIC", "dlq.dispatch.email")
	cfg.ProviderEventsTopic = getEnv("PROVIDER_EVENTS_TOPIC", "provider.events")

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid value for %s: %w", key, err)
		}
		return parsed, nil
	}
	return fallback, nil
}
