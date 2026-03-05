# TGPlane

Distributed Telegram session manager built on TDLib. Designed to run up to 25 000 concurrent account and bot sessions across multiple worker nodes.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Main Node                          │
│                                                         │
│  REST API (:8080)  ──►  Session Pool  ──►  Worker Mgr  │
│       │                                        │        │
│  PostgreSQL + Redis Streams + Webhooks         │ gRPC   │
└────────────────────────────────────────────────┼────────┘
                                                 │
              ┌──────────────────────────────────┤
              │                                  │
     ┌────────▼────────┐              ┌──────────▼───────┐
     │   Worker Node   │              │   Worker Node    │
     │  TDLib sessions │              │  TDLib sessions  │
     │  gRPC (:50051)  │              │  gRPC (:50051)   │
     └─────────────────┘              └──────────────────┘
```

**Main node** — REST API, PostgreSQL, Redis, webhook dispatcher, worker registry.
**Worker node** — TDLib session pool, gRPC server, Prometheus metrics.
Communication: main ↔ worker via gRPC (protobuf).
Updates: TDLib → worker → Redis Stream (`tgplane:updates`) → webhooks.

## Requirements

| Dependency | Version |
|---|---|
| Go | 1.25+ |
| PostgreSQL | 14+ |
| Redis | 7+ |
| TDLib | 1.8.x (built from submodule) |
| cmake / gcc / gperf / openssl | build deps for TDLib |

## Quick Start

### 1. Clone

```bash
git clone https://github.com/tgplane/tgplane
cd tgplane
```

### 2. Install Go (if not installed)

```bash
bash scripts/install-go.sh
```

### 3. Build TDLib

The script clones TDLib at the commit required by go-tdlib v0.7.6 (version 1.8.62), installs build dependencies, compiles, and installs to `/usr/local`. Takes ~10 minutes.

```bash
bash scripts/build-tdlib.sh
```

To use a specific commit or an existing source tree:

```bash
TDLIB_COMMIT=22d49d5b87a4d5fc60a194dab02dd1d71529687f bash scripts/build-tdlib.sh
TDLIB_DIR=/path/to/td bash scripts/build-tdlib.sh
```

Supported systems: **Arch / CachyOS**, **Debian / Ubuntu**, **Fedora / RHEL**, **macOS**.

### 4. Run project setup

```bash
bash scripts/setup.sh
```

Creates `config.yaml` and `config.worker.yaml` from examples, downloads Go modules.

### 5. Start infrastructure

```bash
docker compose -f deployments/docker-compose.yml up -d
```

### 6. Configure

Edit the configs created by `setup.sh`:

```yaml
# config.yaml — main node
database:
  dsn: "postgres://tgplane:tgplane@localhost:5432/tgplane?sslmode=disable"
auth:
  master_key: "your-secret-master-key"

# config.worker.yaml — worker node
tdlib:
  api_id: 123456       # from https://my.telegram.org
  api_hash: "abc..."
```

### 7. Run migrations

```bash
make migrate-up
```

### 8. Run

```bash
# Terminal 1 — main node
make run-main

# Terminal 2 — worker node
make run-worker
```

Or build binaries:

```bash
make build-main
make build-worker          # requires TDLib headers in /usr/local
```

## Configuration

### Main node (`config.yaml`)

```yaml
app:
  mode: main
  name: tgplane

database:
  dsn: "postgres://user:pass@localhost:5432/tgplane?sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime_seconds: 300

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

grpc:
  listen_addr: ":50051"

http:
  addr: ":8080"

auth:
  master_key: ""        # leave empty to disable master key

rate_limit:
  rps: 100
  burst: 200

log:
  level: info           # debug | info | warn | error
  json: false
```

### Worker node (`config.worker.yaml`)

```yaml
app:
  mode: worker
  name: tgplane-worker-1

tdlib:
  api_id: 0             # from https://my.telegram.org
  api_hash: ""
  data_dir: "./data/sessions"
  log_level: 1          # 0=off, 1=error, 2=warn, 3=info, 4=debug
  use_test_dc: false

grpc:
  main_addr: "localhost:50051"

http:
  addr: ":8081"

log:
  level: info
  json: false
```

## REST API

Base URL: `http://localhost:8080/api/v1`
Auth: `X-Api-Key: <key>` header required on all endpoints except `/auth/keys`.

### API Keys

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/keys` | Create API key (returns raw key once) |
| `GET` | `/auth/keys` | List API keys |
| `DELETE` | `/auth/keys/:id` | Delete API key |

Bootstrap first key using `auth.master_key` from config:
```bash
curl -X POST http://localhost:8080/api/v1/auth/keys \
  -H "X-Api-Key: <master_key>" \
  -H "Content-Type: application/json" \
  -d '{"name": "admin"}'
```

### Accounts (user sessions)

| Method | Path | Description |
|---|---|---|
| `POST` | `/accounts` | Add account `{"phone": "+79001234567"}` |
| `GET` | `/accounts` | List accounts |
| `GET` | `/accounts/:id` | Get account |
| `DELETE` | `/accounts/:id` | Delete account |

### Bots

| Method | Path | Description |
|---|---|---|
| `POST` | `/bots` | Add bot `{"token": "123:ABC"}` |
| `GET` | `/bots` | List bots |
| `GET` | `/bots/:id` | Get bot |
| `DELETE` | `/bots/:id` | Delete bot |

### Sessions

| Method | Path | Description |
|---|---|---|
| `GET` | `/sessions` | List all active sessions |
| `GET` | `/sessions/:id` | Get session |
| `DELETE` | `/sessions/:id` | Stop session |

### Workers

| Method | Path | Description |
|---|---|---|
| `GET` | `/workers` | List registered workers |
| `GET` | `/workers/metrics` | Collect metrics from all workers |
| `POST` | `/workers` | Register worker `{"id": "w1", "addr": "host:50051"}` |
| `DELETE` | `/workers/:id` | Unregister worker |
| `POST` | `/workers/:id/drain` | Migrate all sessions off worker |

### Webhooks

| Method | Path | Description |
|---|---|---|
| `POST` | `/webhooks` | Register webhook `{"url": "...", "secret": "...", "events": ["message"]}` |
| `GET` | `/webhooks` | List webhooks |
| `DELETE` | `/webhooks/:id` | Delete webhook |

Webhook payload is signed with `X-Signature: sha256=<hmac>` using the registered secret.
Delivery: 3 retries with 1 s backoff.

### Bulk

| Method | Path | Description |
|---|---|---|
| `POST` | `/bulk/bots` | Add up to 500 bots at once |
| `POST` | `/bulk/accounts` | Add up to 500 accounts at once |
| `DELETE` | `/bulk/sessions` | Remove up to 500 sessions at once |

Response: HTTP 207 Multi-Status with per-item results.

### System

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness check |
| `GET` | `/ready` | Readiness check |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/ui` | Web UI |

## Production Deployment

```bash
# Start full stack
make prod-up

# View logs
make prod-logs

# Stop
make prod-down
```

See `deployments/docker-compose.prod.yml` for the full configuration.
Prometheus scrapes `:8080/metrics` and `:8081/metrics`.
Grafana is available at `http://localhost:3000`.

## Development

```bash
# Run all unit tests
make test

# Run integration tests (requires Docker)
make test-integration

# Run benchmarks (1 min)
make bench

# Run linter
make lint

# Regenerate protobuf
make proto
```

### Project layout

```
cmd/
  main/       — main node entrypoint
  worker/     — worker node entrypoint
internal/
  account/    — user account domain
  auth/       — API key management
  bot/        — bot domain
  bulk/       — bulk operations
  config/     — configuration loading
  logger/     — zap logger factory
  metrics/    — Prometheus metrics
  redisclient/— Redis client factory
  replication/— message replication to PostgreSQL
  session/    — TDLib session pool (CGO-free interface)
  stream/     — Redis Streams publisher
  tdlib/      — TDLib CGO wrapper
  webhook/    — webhook service and dispatcher
  worker/
    manager/  — worker registry and session assignment
    server/   — gRPC worker server
api/
  proto/      — protobuf definitions
  rest/       — Gin HTTP handlers and middleware
migrations/   — SQL migrations (golang-migrate)
deployments/  — Docker Compose, Prometheus config
scripts/      — TDLib build script
web/          — React + TypeScript + Tailwind UI
```

## Benchmarks

Measured on Intel Xeon E5-2620 v3 @ 2.40GHz, 12 cores.

| Operation | ns/op | Notes |
|---|---|---|
| `Auth.Validate` master key | 5 ns | 0 alloc, string compare |
| `Auth.Validate` DB key | 1 278 ns | SHA-256 + map lookup |
| `Auth.Validate` parallel (×12) | 207 ns | |
| `Pool.Get` (1 000 sessions) | 120 ns | read lock, 2 alloc |
| `Pool.Add + Remove` | 3 200 ns | goroutine spawn |
| `Pool.List` (1 000 sessions) | 179 µs | full snapshot |
| `KeyRateLimiter` (1–1 000 keys) | ~480 ns | stable, no degradation |
| `Auth middleware` (full HTTP) | 3 200 ns | |
| `Manager.AssignBot` (3 workers) | 2.7 ms | gRPC round-trip |

## License

MIT
