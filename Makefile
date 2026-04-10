.PHONY: build run test test-short test-integration coverage lint vet staticcheck mocks \
	migrate-up migrate-down migrate-create sqlc docker-up docker-down tidy

# ── Build ────────────────────────────────────────────────────
build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

run:
	go run ./cmd/api

worker:
	go run ./cmd/worker

# ── Test ─────────────────────────────────────────────────────
test:
	go test ./... -race

test-short:
	go test ./... -race -short

test-integration:
	go test ./internal/repository/... -race -v -count=1

coverage:
	go test ./... -race -short -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -1
	@echo "Open coverage report: go tool cover -html=coverage.out"

# ── Lint / Vet ───────────────────────────────────────────────
lint:
	golangci-lint run

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

# ── Code Generation ──────────────────────────────────────────
sqlc:
	sqlc generate

mocks:
	mockery

# ── Database ─────────────────────────────────────────────────
migrate-up:
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-create:
	goose -dir db/migrations create $(name) sql

# ── Docker ───────────────────────────────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-logs:
	docker compose logs -f

# ── Misc ─────────────────────────────────────────────────────
tidy:
	go mod tidy

check: vet staticcheck test-short
	@echo "All checks passed."
