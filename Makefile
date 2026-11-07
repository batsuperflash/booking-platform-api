DC ?= docker-compose
POSTGRES_CONTAINER ?= booking-platform-postgres

.PHONY: up down restart logs ps migrate run test test-storage fmt tidy clean

up:
	$(DC) up --build -d

down:
	$(DC) down

restart:
	$(DC) down
	$(DC) up --build -d

logs:
	$(DC) logs -f

ps:
	$(DC) ps

migrate:
	docker exec -i $(POSTGRES_CONTAINER) psql -U booking -d booking < migrations/001_create_tables.sql

run:
	go run ./cmd/api

test:
	go test ./...

test-storage:
	go test ./internal/storage -run TestConcurrentBookingSameSlot -v

fmt:
	gofmt -w cmd internal

tidy:
	go mod tidy

clean:
	$(DC) down --remove-orphans