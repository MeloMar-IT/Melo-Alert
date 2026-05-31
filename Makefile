.PHONY: run test lint docker-build migrate-up migrate-down

APP_NAME=signalhub
DOCKER_IMAGE=signalhub:latest

run:
	go run cmd/signalhub/main.go

test:
	go test -v ./...

lint:
	golangci-lint run ./... || go vet ./...

docker-build:
	docker build -t $(DOCKER_IMAGE) .

migrate-up:
	@echo "Migrating up..."
	# Placeholder for migration tool like golang-migrate
	# migrate -path migrations -database "$(DATABASE_DSN)" up

migrate-down:
	@echo "Migrating down..."
	# migrate -path migrations -database "$(DATABASE_DSN)" down
