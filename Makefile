.PHONY: build run test lint migrate-up migrate-down migrate-create sqlc docker-up docker-down

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

run:
	go run ./cmd/api

worker:
	go run ./cmd/worker

test:
	go test ./... -v -race

lint:
	golangci-lint run

migrate-up:
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-create:
	goose -dir db/migrations create $(name) sql

sqlc:
	sqlc generate

docker-up:
	docker compose up -d

docker-down:
	docker compose down

tidy:
	go mod tidy
