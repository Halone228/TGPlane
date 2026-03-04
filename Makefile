.PHONY: build build-main build-worker run-main run-worker infra-up infra-down tidy lint test

proto:
	protoc \
		--go_out=. --go_opt=module=github.com/tgplane/tgplane \
		--go-grpc_out=. --go-grpc_opt=module=github.com/tgplane/tgplane \
		-I api/proto \
		api/proto/worker.proto

build: build-main build-worker

build-main:
	go build -o bin/tgplane-main ./cmd/main

build-worker:
	CGO_LDFLAGS_ALLOW="-Wl,--whole-archive.*|-Wl,--no-whole-archive" go build -o bin/tgplane-worker ./cmd/worker

run-main:
	go run ./cmd/main --config config.yaml

run-worker:
	CGO_LDFLAGS_ALLOW="-Wl,--whole-archive.*|-Wl,--no-whole-archive" go run ./cmd/worker --config config.worker.yaml

infra-up:
	docker compose -f deployments/docker-compose.yml up -d

infra-down:
	docker compose -f deployments/docker-compose.yml down

prod-up:
	docker compose -f deployments/docker-compose.prod.yml up -d --build

prod-down:
	docker compose -f deployments/docker-compose.prod.yml down

prod-logs:
	docker compose -f deployments/docker-compose.prod.yml logs -f

migrate-up:
	bash scripts/migrate.sh up

migrate-down:
	bash scripts/migrate.sh down 1

migrate-version:
	bash scripts/migrate.sh version

setup:
	bash scripts/setup.sh

build-tdlib:
	bash scripts/build-tdlib.sh

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

bench:
	go test -run='^$$' -bench=. -benchtime=3s -benchmem \
		./internal/auth/... \
		./internal/session/... \
		./internal/worker/manager/... \
		./api/rest/middleware/...

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 \
		./internal/session/... \
		./internal/account/... \
		./internal/bot/... \
		./internal/auth/... \
		./internal/bulk/... \
		./internal/metrics/... \
		./internal/webhook/... \
		./internal/worker/server/... \
		./internal/worker/manager/... \
		./api/rest/handler/... \
		./api/rest/middleware/...

test-integration:
	go test -race -count=1 -tags integration -timeout 120s \
		./internal/account/... \
		./internal/bot/...

test-all: test test-integration
