# High-Scale Notification Platform (HSNP)

This repository contains the Phase I implementation of the High-Scale Notification Platform. It includes:

- **Ingestion Service (Go)** – Validates API requests, enforces idempotency, persists metadata to Postgres, and publishes to Kafka.
- **Dispatcher (Go)** – Consumes the ingress topic and routes notifications to per-channel topics.
- **Email Worker (Go)** – Pulls from the email dispatch topic, fails over between SES and SendGrid adapters, emits provider events, and writes to a DLQ on exhaustion.
- **Webhook Service (Go)** – Normalizes provider callbacks and publishes canonical events back to Kafka.
- **Admin UI (Next.js)** – Provides dashboards and CRUD entry points for tenants and templates.

Refer to [docs/architecture.md](docs/architecture.md) for the full system blueprint.

## Getting started

All Go services rely on Postgres and Kafka. The expected environment variables are documented inline in each `cmd/<service>/main.go` entrypoint. A minimal local setup requires:

```bash
export DATABASE_URL=postgres://hsnp:hsnp@localhost:5432/hsnp?sslmode=disable
export KAFKA_BROKERS=localhost:9092
```

Each service exposes Prometheus metrics on `METRICS_PORT` (default HTTP port + 1000) and emits traces via OTLP when `OTLP_ENDPOINT` is provided.

Install Go dependencies (once network access is available) with:

```bash
go mod tidy
```

Run a service with:

```bash
go run ./cmd/ingestion
```

or build binaries using `go build ./cmd/<service>`.

### Admin UI

Install dependencies and start the UI:

```bash
cd admin
npm install
npm run dev
```

The UI will be available at `http://localhost:3000`.
