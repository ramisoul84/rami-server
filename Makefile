.PHONY: docker-build docker-run docker-up docker-down docker-logs docker-clean

# Build the Docker image
run:
	go run cmd/server/main.go

# Build the Docker image
docker-build:
	docker build -t rami-server:latest .

# Build and start the full stack (Postgres + app)
docker-up:
	docker compose up -d --build

# Stop and remove containers (data persists in volumes)
docker-down:
	docker compose down

# Stop and remove everything, including the Postgres volume
docker-clean:
	docker compose down -v

# Tail the app logs
docker-logs:
	docker compose logs -f app

# Open a shell inside the app container
docker-shell:
	docker compose exec app sh

# Open a psql session against the Postgres container
docker-psql:
	docker compose exec postgres psql -U postgres -d analytics