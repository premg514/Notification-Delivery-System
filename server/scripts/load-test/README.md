# Load Testing Suite

This folder contains production-oriented load testing assets for the Notification Delivery System.

## Folder Structure

```
server/scripts/load-test/
  README.md
  data/
    users.example.json
  k6/
    send_notification.js
  locust/
    locustfile.py
    requirements.txt
  sql/
    export_users.sql
```

## What This Tests

- API throughput for `POST /send-notification`
- High concurrency behavior
- Queue spike behavior
- Error-rate and latency thresholds

## Prerequisites

1. API, Redis, RabbitMQ, and Postgres are running.
2. You have user UUIDs in the `users` table.
3. Rate limiting is tuned for load tests.

Recommended for load tests:

- Increase `RATE_LIMIT_PER_MINUTE` significantly (or use a dedicated load-test environment).
- Scale worker counts (`FANOUT_WORKER_COUNT`, `DELIVERY_WORKER_COUNT`) to realistic values.

## Prepare User Data

Seed users if needed:

```powershell
cd server
go run ./cmd/seed
```

Export UUIDs into `server/scripts/load-test/data/users.json`:

```powershell
psql "$env:POSTGRES_URL" -At -f server/scripts/load-test/sql/export_users.sql > server/scripts/load-test/data/users.json
```

`users.json` must be a JSON array of UUID strings, for example:

```json
["11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"]
```

## Run k6

Install k6, then run:

```powershell
cd server/scripts/load-test/k6
k6 run send_notification.js -e BASE_URL=http://localhost:8080 -e SCENARIO=baseline -e TARGET_USERS_PER_REQUEST=50 -e TEST_RUN_ID=run1
```

Supported scenarios:

- `smoke`
- `baseline`
- `spike`
- `soak`

Useful env vars:

- `BASE_URL` (default: `http://localhost:8080`)
- `SCENARIO` (default: `baseline`)
- `TARGET_USERS_PER_REQUEST` (default: `50`)
- `TEST_RUN_ID` (default: `local`)
- `MAX_RESPONSE_MS` (default: `1000`)
- `MAX_ERROR_RATE` (default: `0.02`)

## Run Locust

Install dependencies:

```powershell
cd server/scripts/load-test/locust
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
```

Headed mode:

```powershell
locust -f locustfile.py --host http://localhost:8080
```

Headless example:

```powershell
locust -f locustfile.py --host http://localhost:8080 --headless --users 200 --spawn-rate 20 --run-time 10m --csv ../data/locust_run
```

Useful env vars:

- `TARGET_USERS_PER_REQUEST` (default: `50`)
- `MIN_WAIT_MS` (default: `50`)
- `MAX_WAIT_MS` (default: `250`)
- `REQUEST_TIMEOUT_SECONDS` (default: `15`)
- `TEST_RUN_ID` (default: `local`)

## Notes

- Every request sends a unique `Idempotency-Key`, so duplicates should be minimal.
- A `200` can still happen if your idempotency policy detects a duplicate key collision.
- For true stress testing (100K+ events), prefer dedicated infrastructure and isolate other workloads.
