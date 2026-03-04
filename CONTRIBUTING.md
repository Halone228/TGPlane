# Contributing to TGPlane

Thank you for your interest in contributing! This document covers the project architecture, development workflow, conventions, and testing guidelines.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Repository Layout](#repository-layout)
3. [Prerequisites](#prerequisites)
4. [Getting Started](#getting-started)
5. [Configuration](#configuration)
6. [Development Workflow](#development-workflow)
7. [Code Conventions](#code-conventions)
8. [Testing](#testing)
9. [Adding a New Feature](#adding-a-new-feature)
10. [REST API Guidelines](#rest-api-guidelines)
11. [gRPC / Protobuf Guidelines](#grpc--protobuf-guidelines)
12. [Database Migrations](#database-migrations)
13. [Commit & PR Guidelines](#commit--pr-guidelines)

---

## Architecture Overview

TGPlane is a multi-node Telegram session manager built on [TDLib](https://core.telegram.org/tdlib).

```
┌─────────────────────────────────────────────────────────┐
│                       Main Node                         │
│                                                         │
│  REST API (Gin)  ←→  Worker Manager  ←→  gRPC clients  │
│       │                    │                            │
│  Auth middleware     Redis Streams publisher            │
│  Bulk API            Webhook Dispatcher                 │
│       │                                                 │
│  PostgreSQL (accounts, bots, api_keys, webhooks)        │
└────────────────────────┬────────────────────────────────┘
                         │ gRPC
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │ Worker 1 │  │ Worker 2 │  │ Worker N │
    │ TDLib    │  │ TDLib    │  │ TDLib    │
    │ sessions │  │ sessions │  │ sessions │
    └──────────┘  └──────────┘  └──────────┘
          │              │
          └──────────────┴──→ Redis Stream "tgplane:updates"
                                      │
                              Webhook Dispatcher
                                      │
                              POST to consumers
```

**Key data flows:**
- **Add bot/account** → REST → `bot/account.Service` → DB → `manager.AssignBot/Account` → gRPC → Worker
- **Telegram update** → TDLib → Worker → gRPC `Subscribe` stream → Main → Redis Stream → Webhook Dispatcher → HTTP POST
- **Bulk operation** → REST → `bulk.Service` (parallel, semaphore-bounded) → same as above per item

---

## Repository Layout

```
TGPlane/
├── api/
│   ├── proto/                  # Protobuf definitions
│   │   ├── worker.proto
│   │   └── gen/tgplane/v1/     # Generated Go code (do not edit)
│   └── rest/
│       ├── handler/            # Gin HTTP handlers
│       ├── middleware/         # Logger, metrics, auth
│       └── server.go           # Server wiring
├── cmd/
│   ├── main/                   # Main node entrypoint
│   └── worker/                 # Worker node entrypoint
├── deployments/
│   ├── docker-compose.yml      # Postgres, Redis, Prometheus, Grafana
│   └── prometheus.yml
├── internal/
│   ├── account/                # Account domain (model, repo, service)
│   ├── auth/                   # API key management
│   ├── bot/                    # Bot domain (model, repo, service)
│   ├── bulk/                   # Bulk operations service
│   ├── config/                 # Viper config
│   ├── database/               # sqlx connect + migrate
│   ├── logger/                 # zap factory
│   ├── metrics/                # Prometheus metric definitions + session hook
│   ├── redisclient/            # Redis client factory
│   ├── session/                # Session pool (CGO-free)
│   ├── stream/                 # Redis Stream publisher
│   ├── tdlib/                  # TDLib CGO wrapper (CGO required)
│   ├── testhelper/             # Shared test utilities
│   ├── webhook/                # Webhook model, repo, service, dispatcher
│   └── worker/
│       ├── client/             # gRPC client (used by main)
│       ├── manager/            # Multi-worker manager
│       └── server/             # gRPC server (runs on worker)
├── migrations/                 # golang-migrate SQL files
├── scripts/
│   └── build-tdlib.sh          # Builds TDLib from the submodule
├── config.yaml                 # Main node config
├── config.worker.yaml          # Worker node config
├── Makefile
└── go.mod
```

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | ≥ 1.22 | Build & test |
| Docker + Compose | any recent | Local infra (Postgres, Redis) |
| protoc | ≥ 25 | Regenerate protobuf (only when `.proto` changes) |
| protoc-gen-go | latest | Go protobuf plugin |
| protoc-gen-go-grpc | latest | Go gRPC plugin |
| golangci-lint | ≥ 1.57 | Linting |
| golang-migrate CLI | ≥ 4.x | Manual migrations (optional, auto-run on startup) |

> **Note:** TDLib (CGO) is only needed for `internal/tdlib` and `cmd/worker`. All other packages compile without it. If you are working on the main node, REST API, or tests, you do not need TDLib headers.

### Install protoc plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Build TDLib (worker only)

```bash
bash scripts/build-tdlib.sh
```

This builds TDLib from the `telegram-database/` submodule and installs headers/libs to `telegram-database/build/`.

---

## Getting Started

```bash
# 1. Clone with submodules
git clone --recurse-submodules https://github.com/tgplane/tgplane.git
cd tgplane

# 2. Start infrastructure
make infra-up
# Postgres :5432, Redis :6379, Prometheus :9090, Grafana :3000

# 3. Copy and edit configs
cp config.yaml config.local.yaml
# Set tdlib.api_id, tdlib.api_hash (from https://my.telegram.org)
# Set auth.master_key to a strong random string

# 4. Run main node (migrations run automatically on startup)
make run-main

# 5. (Optional) Run a worker node
make run-worker
```

After startup:
- REST API: `http://localhost:8080`
- Prometheus metrics: `http://localhost:8080/metrics`
- Health: `http://localhost:8080/health`

### Bootstrap an API key

The `/api/v1/auth/keys` endpoint is unprotected for bootstrapping:

```bash
curl -X POST http://localhost:8080/api/v1/auth/keys \
  -H "Content-Type: application/json" \
  -d '{"name": "my-first-key"}'
# Response includes "key" — save it, shown only once.

# Or use master_key from config.yaml directly:
curl http://localhost:8080/api/v1/bots \
  -H "X-Api-Key: <master_key>"
```

---

## Configuration

### Main node (`config.yaml`)

```yaml
database:
  dsn: "postgres://tgplane:tgplane@localhost:5432/tgplane?sslmode=disable"

redis:
  addr: "localhost:6379"

auth:
  master_key: ""          # optional bypass key; empty = DB-only auth

tdlib:
  api_id: 12345
  api_hash: "abc123"
  data_dir: "./data/sessions"

http:
  addr: ":8080"

log:
  level: info             # debug | info | warn | error
  json: false             # true in production
```

### Worker node (`config.worker.yaml`)

```yaml
app:
  name: tgplane-worker-1  # unique name per worker

tdlib:
  api_id: 12345
  api_hash: "abc123"

grpc:
  main_addr: "localhost:50051"

http:
  addr: ":8081"           # exposes /metrics
```

---

## Development Workflow

```bash
make test           # unit tests with race detector
make test-integration # integration tests (needs Docker)
make test-all       # both

make lint           # golangci-lint
make tidy           # go mod tidy
make proto          # regenerate protobuf (after editing .proto)
make build          # build both binaries to bin/
make infra-up       # start Postgres + Redis + monitoring
make infra-down     # stop
```

---

## Code Conventions

### Package structure

Each domain package (`account`, `bot`, `auth`, `webhook`) follows the same layout:

```
model.go        — structs and constants
repository.go   — Repository interface
postgres.go     — PostgreSQL implementation
memory_repo.go  — In-memory implementation (tests only)
service.go      — Business logic
*_test.go       — Unit tests using memory_repo
```

### CGO isolation

`internal/tdlib` is the **only** package that imports `go-tdlib` (CGO). All other packages use the `session.TDClient` interface. This keeps unit tests CGO-free and fast.

### Error handling

- Return `error` from functions; wrap with `fmt.Errorf("context: %w", err)`.
- Domain-specific sentinel errors (e.g., `auth.ErrNotFound`, `webhook.ErrNotFound`) live in `model.go`.
- HTTP handlers translate errors to appropriate status codes — do not return raw DB errors to clients.

### Logging

Use `go.uber.org/zap`. Never use `fmt.Print*` or the standard `log` package in production code.

```go
// good
log.Info("bot registered", zap.Int64("id", b.ID))
log.Error("factory failed", zap.Error(err))

// bad
fmt.Println("bot registered:", b.ID)
```

### Context

Every function that does I/O must accept `context.Context` as its first parameter. Never use `context.Background()` inside library code — only in `main` and top-level goroutines.

### Concurrency

- The `session.Pool` is safe for concurrent use.
- New shared state must be protected with `sync.RWMutex` (prefer RLock for reads).
- Use semaphore pattern (`chan struct{}`) for bounded parallelism (see `bulk.Service`).

---

## Testing

### Unit tests

All unit tests are in `_test.go` files and use in-memory repositories. They must run without Docker, network, or CGO:

```bash
make test
# or directly:
go test -race -count=1 ./internal/...  ./api/rest/...
```

### Integration tests (build tag `integration`)

Integration tests use [testcontainers-go](https://golang.testcontainers.org/) to spin up a real PostgreSQL container:

```bash
make test-integration
# or:
go test -race -count=1 -tags integration -timeout 120s ./internal/account/... ./internal/bot/...
```

Each integration test file starts with:
```go
//go:build integration
```

Use `TestMain` + a shared container + `TRUNCATE … RESTART IDENTITY CASCADE` between tests — do **not** spin up a new container per test.

### Writing a new test

1. **Unit test** — always prefer. Use `MemoryRepository` implementations.
2. **gRPC test** — use `net.Listen("tcp", "127.0.0.1:0")` for an in-process server (no Docker needed). See `internal/worker/server/server_test.go`.
3. **HTTP handler test** — use `httptest.NewRecorder()` + `gin.TestMode`.
4. **Integration test** — use `testcontainers-go` + `//go:build integration` tag.

### Test helpers

| Helper | Location | Purpose |
|--------|----------|---------|
| `account.NewMemoryRepository()` | `internal/account/memory_repo.go` | In-memory accounts |
| `bot.NewMemoryRepository()` | `internal/bot/memory_repo.go` | In-memory bots |
| `auth.NewMemoryRepository()` | `internal/auth/memory_repo.go` | In-memory API keys |
| `webhook.NewMemoryRepository()` | `internal/webhook/memory_repo.go` | In-memory webhooks |
| `testhelper.NewPostgresDB()` | `internal/testhelper/postgres.go` | Shared PG container |

---

## Adding a New Feature

### 1. New domain entity (e.g., `Channel`)

```
internal/channel/
  model.go          — Channel struct, constants, sentinel errors
  repository.go     — Repository interface
  postgres.go       — PostgreSQL impl (sqlx)
  memory_repo.go    — In-memory impl for tests
  service.go        — Business logic
  service_test.go   — Unit tests
```

Add a migration:
```bash
# create files migrations/000004_channels.up.sql and .down.sql
```

### 2. New REST endpoint

1. Add handler in `api/rest/handler/<entity>.go`.
2. Register routes in the handler's `Register(r gin.IRouter)` method.
3. Wire in `api/rest/server.go` (add to `Deps` if needed).
4. Add handler tests in `api/rest/handler/<entity>_test.go`.

### 3. New gRPC method

1. Add the RPC and messages to `api/proto/worker.proto`.
2. Run `make proto` to regenerate Go code.
3. Implement the method in `internal/worker/server/server.go`.
4. Add the client method in `internal/worker/client/client.go`.
5. Expose via manager in `internal/worker/manager/manager.go` if needed.

### 4. New Prometheus metric

Add to `internal/metrics/metrics.go` using `promauto`:

```go
var MyCounter = promauto.NewCounterVec(prometheus.CounterOpts{
    Namespace: "tgplane",
    Subsystem: "my_subsystem",
    Name:      "things_total",
    Help:      "Total number of things.",
}, []string{"label"})
```

---

## REST API Guidelines

- Base path: `/api/v1/`
- All endpoints (except `/health`, `/ready`, `/metrics`, `/api/v1/auth/keys`) require `X-Api-Key` header.
- Response format for single resource: JSON object.
- Response format for list: JSON array.
- Bulk operations return `207 Multi-Status` with `{"total", "succeeded", "failed", "items": [...]}`.
- Error response: `{"error": "message"}`.
- Pagination via `?limit=50&offset=0` query params.

### Status codes

| Situation | Code |
|-----------|------|
| Created | 201 |
| Updated / no body | 204 |
| Validation error | 400 |
| Unauthorized | 401 |
| Not found | 404 |
| Bulk (partial) | 207 |
| Server error | 500 |

---

## gRPC / Protobuf Guidelines

- Proto file: `api/proto/worker.proto`.
- Generated code lives in `api/proto/gen/tgplane/v1/` — **do not edit by hand**.
- After editing `.proto`, run `make proto` to regenerate.
- Use `google.protobuf.Timestamp` for timestamps in new messages.
- All new RPCs must be covered by a test in `internal/worker/server/server_test.go`.

---

## Database Migrations

Migrations are in `migrations/` and applied automatically on main node startup via `golang-migrate`.

Naming convention: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`

```
000001_init.up.sql          — accounts, bots tables
000002_api_keys.up.sql      — api_keys table
000003_webhooks.up.sql      — webhooks table
000004_...                  — your next migration
```

Rules:
- Each migration must have a corresponding `.down.sql`.
- `.down.sql` must fully reverse `.up.sql`.
- Never modify an already-merged migration — always add a new one.
- Use `BIGSERIAL` for primary keys, `TIMESTAMPTZ` for timestamps.

---

## Commit & PR Guidelines

### Commit messages

Follow Conventional Commits:

```
feat: add channel management endpoints
fix: prevent nil pointer in session pool on factory error
refactor: extract semaphore helper to bulk package
test: add integration test for bot postgres repo
docs: add CONTRIBUTING.md
```

### Pull Request checklist

- [ ] `make test` passes with `-race`
- [ ] `make lint` passes (zero warnings)
- [ ] New code has unit tests (≥ happy path + error path)
- [ ] New DB columns/tables have a migration with `.down.sql`
- [ ] New config fields are documented in this guide and have defaults in `setDefaults()`
- [ ] No secrets or credentials committed
- [ ] Proto changes run through `make proto` and generated files are committed
