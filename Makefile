.PHONY: test test-unit test-integration test-host tidy fmt vet migrate run

# Default: the containerized unit suite, mirroring rdt's `tests unit`.
test: test-unit

# Unit tests inside the app container (Go's -short skips anything that needs a
# live service, the idiomatic stand-in for rdt's `not integration_test` marker).
test-unit:
	docker compose build app
	docker compose run --rm app go test -short ./...

# Full suite including integration tests: bring up Postgres, wait for it to be
# healthy, then run without -short so service-backed tests run. (redis/minio
# join here as the modules that need them are ported.)
test-integration:
	docker compose up -d --wait postgres
	docker compose build app
	docker compose run --rm app go test ./...

# Host-side convenience (no container) — fast inner loop while developing.
test-host:
	go test ./...

# Apply schema migrations in the app container against the compose Postgres.
# Migrations are an explicit step now (the api no longer migrates on boot), so
# run this before `make`-ing anything that needs the schema. Override the goose
# command with CMD, e.g. `make migrate CMD=status` or `make migrate CMD=down`.
migrate: CMD ?= up
migrate:
	docker compose up -d --wait postgres
	docker compose build app
	docker compose run --rm app go run ./cmd/migrate $(CMD)

# Run the api server in the compose network and publish it to the host. Applies
# migrations first (via the `migrate` target — the api no longer migrates on
# boot), then serves in the foreground; Ctrl-C stops it. Default host port 8080;
# override with API_PORT=... (e.g. `make run API_PORT=8090`) if it clashes.
run: migrate
	docker compose up --build api

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...
