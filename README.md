# Notification Delivery System

An asynchronous department-based notification platform with a Next.js frontend and a Go backend. Operators can submit a message for a department such as `CSE`, `ECE`, `ME`, `CIVIL`, or `EEE`; the API stores the notification, queues fanout work in RabbitMQ, creates delivery records in PostgreSQL, and processes delivery attempts with retry support.

## What This Repo Contains

- `client/`: operator dashboard and history UI built with Next.js, React, TypeScript, and Tailwind CSS
- `server/`: Go services for HTTP API, workers, queueing, persistence, rate limiting, and metrics

## Architecture At A Glance

```text
Browser UI
  -> Next.js route handlers
  -> Go API
  -> PostgreSQL (notifications + deliveries + users)
  -> RabbitMQ fanout queue
  -> RabbitMQ delivery queue
  -> worker service
  -> retry queues with delay
  -> delivery status updates in PostgreSQL
```

## End-To-End Flow

### Notification send flow

1. The dashboard form in `client/components/dashboard.tsx` collects title, message, department, and priority.
2. The client generates an idempotency key and sends `POST /api/notifications`.
3. `client/app/api/notifications/route.ts` proxies the request to `POST /send-notification` on the Go API.
4. `server/internal/api/handlers/notification.go` validates the payload and the `Idempotency-Key` header.
5. `server/internal/service/notification_service.go` looks up users in the target department, normalizes priority, and inserts the notification idempotently.
6. If the notification is new, `server/internal/queue/publisher.go` publishes a `FanoutJob` to RabbitMQ.
7. `server/internal/worker/fanout_processor.go` consumes that job, creates one delivery row per target user, and publishes `DeliveryBatchJob` messages.
8. `server/internal/worker/delivery_processor.go` consumes delivery batches, simulates delivery via `LoggingSender`, and updates delivery status.
9. Failed delivery attempts are pushed to delayed retry queues and re-enter the main delivery queue after TTL expiration.

### Notification history flow

1. The dashboard and history page call `GET /api/notifications/recent`.
2. `client/app/api/notifications/recent/route.ts` proxies the request to the backend.
3. `server/internal/repository/postgres/notification_repo.go` aggregates notification and delivery information from PostgreSQL.
4. The service derives a user-facing status of `process`, `queue`, or `sent`.

## Main Features

- department-targeted notification broadcast
- idempotent notification submission
- asynchronous fanout and delivery processing
- Redis-backed API rate limiting
- Prometheus metrics endpoint
- searchable frontend notification history
- seed script for sample users
- load-testing assets for k6 and Locust

## Tech Stack

### Client

- Next.js 14
- React 18
- TypeScript
- Tailwind CSS

### Server

- Go 1.25
- Gin
- PostgreSQL
- RabbitMQ
- Redis
- Prometheus Go client

## Quick Start

### 1. Start infrastructure

```powershell
docker compose -f server/scripts/docker-compose.yml up -d
```

This starts:

- PostgreSQL on `localhost:5432`
- RabbitMQ on `localhost:5672`
- RabbitMQ management UI on `localhost:15672`
- Redis on `localhost:6379`

RabbitMQ default credentials:

- username: `admin`
- password: `admin`

### 2. Set backend environment variables

Create a `.env` file in the repo root or `server/.env` with:

```env
POSTGRES_URL=postgres://admin:admin@localhost:5432/notification?sslmode=disable
RABBITMQ_URL=amqp://admin:admin@localhost:5672/
REDIS_URL=redis://localhost:6379/0
PORT=8080
LOG_LEVEL=info
RATE_LIMIT_PER_MINUTE=120
```

### 3. Run database migrations automatically

Migrations are applied on startup by both the API and worker processes. No separate migration command is required.

### 4. Seed users

```powershell
cd server
go run ./cmd/seed
```

This inserts 5,000 sample users distributed across all supported departments.

### 5. Run the API

```powershell
cd server
go run ./cmd/api
```

API base URL:

- `http://localhost:8080`

### 6. Run the worker service

```powershell
cd server
go run ./cmd/worker
```

### 7. Run the frontend

```powershell
cd client
npm install
npm run dev
```

Frontend URL:

- `http://localhost:3000`

## API Reference

### `GET /health`

Returns:

```json
{
  "status": "ok"
}
```

### `GET /metrics`

Prometheus metrics endpoint.

### `POST /send-notification`

Headers:

- `Content-Type: application/json`
- `Idempotency-Key: <unique-value>`

Request body:

```json
{
  "title": "Placement Cell Update",
  "message": "Students report to seminar hall at 2 PM",
  "target_department": "CSE",
  "priority": "high"
}
```

Validation rules:

- `title` required
- `message` required
- `target_department` must be one of `CSE`, `ECE`, `ME`, `CIVIL`, `EEE`
- `Idempotency-Key` required
- `priority` normalizes to `low`, `normal`, or `high`

Typical success response:

```json
{
  "notification_id": "uuid",
  "status": "queue",
  "queued_deliveries": 1000,
  "target_department": "CSE",
  "duplicate": false,
  "created_at": "2026-03-18T00:00:00Z"
}
```

Duplicate response:

```json
{
  "notification_id": "uuid",
  "status": "already_sent",
  "queued_deliveries": 1000,
  "target_department": "CSE",
  "duplicate": true,
  "created_at": "2026-03-18T00:00:00Z"
}
```

Possible error responses:

- `400` invalid payload or missing fields
- `400` unsupported department
- `400` no users found for target department
- `429` rate limit exceeded
- `503` rate limiter unavailable
- `500` internal server error

### `GET /notifications/recent?limit=10`

Returns recent notifications with derived status and queued delivery count.

Example response:

```json
[
  {
    "id": "uuid",
    "title": "Placement Cell Update",
    "target_department": "CSE",
    "priority": "high",
    "status": "queue",
    "created_at": "2026-03-18T00:00:00Z",
    "queued_deliveries": 1000
  }
]
```

## Queueing And Retry Behavior

RabbitMQ topology is created by `server/internal/queue/rabbitmq.go`.

Queues:

- fanout queue
- delivery queue
- `notifications.delivery.retry.1`
- `notifications.delivery.retry.2`
- `notifications.delivery.retry.3`

Characteristics:

- all queues are quorum queues
- retry queues use message TTL
- expired retry messages dead-letter back into the main delivery queue

Retry delays:

- attempt 1: `5s`
- attempt 2: `20s`
- attempt 3: `60s`

The delivery processor marks a delivery:

- `sent` if sender succeeds
- `pending` when scheduled for retry
- `failed` if retries exceed `MAX_RETRIES`

## Data Model

### `users`

Stores recipients.

Columns:

- `id`
- `email`
- `device_token`
- `department`
- `created_at`

### `notifications`

Stores one logical notification submission.

Columns:

- `id`
- `title`
- `message`
- `target_department`
- `priority`
- `idempotency_key`
- `created_at`

### `deliveries`

Stores one user-delivery record per notification.

Columns:

- `id`
- `user_id`
- `notification_id`
- `status`
- `retry_count`
- `delivered_at`
- `last_error`
- `updated_at`

Important constraint:

- unique `(notification_id, user_id)` prevents duplicate delivery rows

## Configuration

Configuration is loaded in `server/internal/config/config.go`.

### Required

- `POSTGRES_URL`
- `RABBITMQ_URL`

### Optional

- `PORT` default `8080`
- `LOG_LEVEL` default `info`
- `REDIS_URL` default `redis://localhost:6379/0`
- `RABBITMQ_FANOUT_QUEUE` default `notifications.fanout`
- `RABBITMQ_DELIVERY_QUEUE` default `notifications.delivery`
- `RABBITMQ_DELIVERY_RETRY_PREFIX` default `notifications.delivery.retry`
- `FANOUT_WORKER_COUNT` default `8`
- `DELIVERY_WORKER_COUNT` default `128`
- `RABBITMQ_FANOUT_PREFETCH_COUNT` default `16`
- `RABBITMQ_DELIVERY_PREFETCH_COUNT` default `256`
- `FANOUT_BATCH_SIZE` default `5000`
- `DELIVERY_BATCH_SIZE` default `500`
- `DELIVERY_ATTEMPTS_PER_BATCH` default `32`
- `RATE_LIMIT_PER_MINUTE` default `120`
- `REQUEST_TIMEOUT_SECONDS` default `15`
- `MAX_RETRIES` default `3`
- `NOTIFICATION_API_BASE_URL` for the frontend proxy, default `http://localhost:8080`

## Observability

Metrics are defined in `server/internal/observability/metrics.go` and exposed via `/metrics`.

Tracked metrics include:

- total HTTP requests
- HTTP request duration
- panic recovery count
- rate limit block count
- rate limiter error count
- worker job count by pool and result
- worker job duration by pool and result
- total scheduled retries

Logging uses `log/slog` with JSON output configured in `server/internal/observability/logger.go`.

## Annotated File Map

This section walks through the meaningful source and config files in the repo.

### Root

- `.gitignore`: ignores environment files, Go build artifacts, frontend build outputs, editor files, and logs.
- `PROJECT_DOCUMENTATION.md`: long-form architecture and project documentation.

### Client files

- `client/README.md`: lightweight client-specific setup notes.
- `client/package.json`: frontend dependencies and scripts for development, build, start, cleaning, and linting.
- `client/package-lock.json`: npm lockfile.
- `client/next.config.mjs`: enables React strict mode.
- `client/next-env.d.ts`: Next.js TypeScript ambient definitions.
- `client/tsconfig.json`: strict TypeScript config with `@/*` path aliasing to the project root of the client app.
- `client/tailwind.config.js`: Tailwind content paths plus custom colors, shadows, radii, and font families used by the UI.
- `client/postcss.config.js`: wires Tailwind CSS and Autoprefixer into the build.
- `client/tsconfig.tsbuildinfo`: TypeScript incremental build cache.

#### `client/app`

- `client/app/layout.tsx`: top-level app shell with metadata and sticky header branding.
- `client/app/page.tsx`: homepage entry that renders the dashboard component.
- `client/app/notifications/page.tsx`: searchable and paginated notification history page.
- `client/app/globals.css`: global Tailwind directives and custom visual system including gradients, tokens, panel styles, and responsive helpers.

#### `client/app/api`

- `client/app/api/health/route.ts`: proxies frontend health checks to backend `/health`.
- `client/app/api/notifications/route.ts`: proxies notification submission to backend `/send-notification`.
- `client/app/api/notifications/recent/route.ts`: proxies recent notification queries to backend `/notifications/recent`.

#### `client/components`

- `client/components/dashboard.tsx`: core UI screen for sending notifications, checking backend health, summarizing activity, and listing recent notifications. It also generates idempotency keys in the browser and periodically refreshes health and activity state.

#### `client/lib`

- `client/lib/api.ts`: typed browser-facing fetch helpers for send, health, and recent notifications.

### Server files

- `server/go.mod`: Go module definition and backend dependencies.
- `server/go.sum`: dependency checksum lockfile.

#### `server/cmd`

- `server/cmd/api/main.go`: API process entrypoint. Loads config, opens PostgreSQL, runs migrations, opens RabbitMQ, creates publisher, builds rate limiter, and starts the Gin HTTP server with graceful shutdown.
- `server/cmd/worker/main.go`: worker process entrypoint. Loads config, opens infrastructure, starts fanout and delivery worker pools, creates RabbitMQ consumers, and runs both pipelines until shutdown.
- `server/cmd/seed/main.go`: seeds 5,000 sample users evenly across supported departments.

#### `server/internal/config`

- `server/internal/config/config.go`: central environment loader. Supports `.env` discovery from several likely paths and exposes tuning values for queue names, worker counts, batch sizes, timeouts, retries, and rate limiting.

#### `server/internal/api`

- `server/internal/api/routes.go`: builds the Gin router, registers middleware, and exposes `/health`, `/metrics`, `POST /send-notification`, and `GET /notifications/recent`.

##### `server/internal/api/handlers`

- `server/internal/api/handlers/notification.go`: validates incoming requests, parses departments, enforces `Idempotency-Key`, delegates to service logic, and maps service errors to HTTP responses.

##### `server/internal/api/middleware`

- `server/internal/api/middleware/request_log.go`: logs request method, path, route, status, duration, and client IP.
- `server/internal/api/middleware/recovery.go`: catches panics, logs them, increments panic metrics, and returns HTTP 500.
- `server/internal/api/middleware/metrics.go`: records per-route request counts and latency metrics.
- `server/internal/api/middleware/ratelimit.go`: Redis-backed fixed-window limiter keyed by client IP and current minute.
- `server/internal/api/middleware/cors.go`: permissive CORS middleware, though it is not currently wired into `routes.go`.

#### `server/internal/domain`

- `server/internal/domain/repository_interface.go`: interface contract used by services and workers for notification and delivery persistence.

##### `server/internal/domain/models`

- `server/internal/domain/models/department.go`: department enum and parser for `CSE`, `ECE`, `ME`, `CIVIL`, and `EEE`.
- `server/internal/domain/models/user.go`: recipient model.
- `server/internal/domain/models/notification.go`: notification model plus aggregated counters used for recent-history views.
- `server/internal/domain/models/delivery.go`: delivery persistence model.
- `server/internal/domain/models/fanout_job.go`: queue payload for fanout work.
- `server/internal/domain/models/delivery_batch_job.go`: queue payload for downstream delivery batches and per-item attempts.

#### `server/internal/repository/postgres`

- `server/internal/repository/postgres/db.go`: PostgreSQL connection helper and filesystem-based SQL migration runner.
- `server/internal/repository/postgres/notification_repo.go`: all SQL access for notifications, user lookup, delivery creation, status updates, idempotent inserts, and recent notification aggregation.

Important repository behaviors:

- `CreateNotificationIfAbsent` uses `ON CONFLICT (idempotency_key)` to provide idempotency.
- `CreateDeliveriesIfAbsent` stages delivery rows in a temporary table, bulk inserts them with `CopyFrom`, and skips duplicates via `ON CONFLICT (notification_id, user_id) DO NOTHING`.
- `ListRecentNotifications` derives queue stats by joining `notifications` and `deliveries` and also falls back to `users` counts when deliveries are not yet present.

#### `server/internal/service`

- `server/internal/service/notification_service.go`: orchestration layer for notification queueing and recent notification formatting.
- `server/internal/service/retry_service.go`: helper for retry delay definitions.
- `server/internal/service/id.go`: UUID generation helper.

Important service behaviors:

- if no users exist for the target department, the request fails before queue publication
- priority is normalized to `low`, `normal`, or `high`
- duplicate idempotency keys return existing notification metadata instead of creating a new record
- recent notification status is derived from queued, pending, and sent counters

#### `server/internal/queue`

- `server/internal/queue/rabbitmq.go`: opens RabbitMQ and declares the queue topology including delayed retry queues.
- `server/internal/queue/publisher.go`: publishes JSON payloads with persistent delivery mode and waits for broker confirmation.
- `server/internal/queue/consumer.go`: generic queue consumer that unmarshals payloads and submits them to worker pools.

#### `server/internal/worker`

- `server/internal/worker/pool.go`: generic worker pool with bounded job channel, per-job timeout, panic safety, and success/failure metrics.
- `server/internal/worker/fanout_processor.go`: expands a fanout job into delivery rows and delivery batch queue messages.
- `server/internal/worker/delivery_processor.go`: processes delivery attempts, updates retry counts and status, and republishes retries when needed.

Important worker behaviors:

- fanout defaults to batches of 5,000 users and delivery sub-batches of 500
- delivery processing defaults to chunks of 32 attempts inside each job
- `LoggingSender` is a placeholder transport that sleeps for 20ms and logs successful sends

#### `server/internal/observability`

- `server/internal/observability/logger.go`: JSON logger initialization with level parsing.
- `server/internal/observability/metrics.go`: Prometheus counters and histograms for HTTP, workers, and retries.

#### `server/migrations`

- `server/migrations/001_create_users.sql`: creates users table and department index.
- `server/migrations/002_create_notifications.sql`: creates notifications table with idempotency key uniqueness.
- `server/migrations/003_create_deliveries.sql`: creates deliveries table, indexes, and unique `(notification_id, user_id)` constraint.

#### `server/scripts`

- `server/scripts/docker-compose.yml`: local infrastructure for PostgreSQL, RabbitMQ, and Redis.

##### `server/scripts/load-test`

- `server/scripts/load-test/README.md`: explains load-test setup and usage.
- `server/scripts/load-test/data/users.json`: sample exported department data used by tests.
- `server/scripts/load-test/k6/send_notification.js`: k6 scenario for notification API load testing.
- `server/scripts/load-test/locust/locustfile.py`: Locust test definition for API load and concurrency testing.
- `server/scripts/load-test/locust/requirements.txt`: Python dependencies for Locust.
- `server/scripts/load-test/locust/__pycache__/locustfile.cpython-311.pyc`: generated Python bytecode cache.
- `server/scripts/load-test/sql/export_users.sql`: SQL helper for exporting department data into JSON for load tests.

## Design Notes And Caveats

- The actual delivery transport is not integrated yet; `LoggingSender` only simulates sends and logs them.
- `server/internal/api/middleware/cors.go` exists but is not currently attached to the router.
- The frontend uses polling for health and recent activity rather than websockets or server-sent events.
- Rate limiting is fixed-window and IP-based, which is fine for simple protection but not a full multi-tenant quota system.
- The history page fetches up to 1,000 recent items and filters them client-side.

## Suggested Next Steps

- replace `LoggingSender` with a real email, SMS, or push-notification provider
- add authentication and authorization
- add backend tests for handlers, services, repository methods, and workers
- expose richer notification analytics and per-delivery breakdowns
- wire CORS middleware if cross-origin browser access is needed
- add a dead-letter queue strategy for exhausted retry cases
