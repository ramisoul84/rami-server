.PHONY: run test lint build up down logs shell

run:            ## Run locally (reads .env)
	go run ./cmd/server

test:           ## Unit tests
	go test ./...

lint:           ## Static checks
	go vet ./...
	test -z "$$(gofmt -l .)"

build:          ## Build the Docker image
	docker compose build

up:             ## Build and (re)start the container
	docker compose up -d --build

down:           ## Stop the container
	docker compose down

logs:           ## Follow the logs
	docker compose logs -f rami-server

shell:          ## Shell inside the container
	docker compose exec rami-server sh
