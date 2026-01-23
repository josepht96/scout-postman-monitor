.PHONY: help build up down logs restart clean test db-connect

# Default target
help:
	@echo "Scout - Postman Test Monitor"
	@echo ""
	@echo "Available commands:"
	@echo "  make build       - Build Docker images"
	@echo "  make up          - Start all services"
	@echo "  make down        - Stop all services"
	@echo "  make logs        - View logs (Ctrl+C to exit)"
	@echo "  make logs-scout  - View Scout logs only"
	@echo "  make logs-db     - View PostgreSQL logs only"
	@echo "  make restart     - Restart all services"
	@echo "  make clean       - Stop services and remove volumes"
	@echo "  make db-connect  - Connect to PostgreSQL database"
	@echo "  make test        - Test Newman executor"
	@echo "  make dev         - Run Scout locally (requires local Postgres)"
	@echo "  make build-go    - Build Scout binary"

# Build Docker images
build:
	docker-compose build

# Start services
up:
	docker-compose up -d
	@echo ""
	@echo "Scout is starting up..."
	@echo "  - Web UI: http://localhost:8080"
	@echo "  - API: http://localhost:8080/api/results"
	@echo "  - Metrics: http://localhost:8080/metrics"
	@echo "  - PostgreSQL: localhost:5432"
	@echo ""
	@echo "Run 'make logs' to view logs"

# Start services in foreground
up-fg:
	docker-compose up

# Stop services
down:
	docker-compose down

# View all logs
logs:
	docker-compose logs -f

# View Scout logs only
logs-scout:
	docker-compose logs -f scout

# View PostgreSQL logs only
logs-db:
	docker-compose logs -f postgres

# Restart services
restart:
	docker-compose restart

# Stop and remove everything including volumes
clean:
	docker-compose down -v
	@echo "Cleaned up all containers and volumes"

# Connect to PostgreSQL
db-connect:
	docker-compose exec postgres psql -U postgres -d scout

# Test Newman executor
test:
	cd newman && node executor.js ../collections/example-api.postman_collection.json

# Run locally (requires local PostgreSQL)
dev:
	@echo "Starting Scout locally..."
	@echo "Make sure PostgreSQL is running on localhost:5432"
	@export DATABASE_URL="postgres://postgres:postgres@localhost:5432/scout?sslmode=disable" && \
	export COLLECTIONS_DIR="./collections" && \
	export INTERVAL="60s" && \
	export PORT="8080" && \
	./scout

# Build Go binary
build-go:
	go build -o scout ./cmd/scout
	@echo "Binary built: scout"

# Install Newman dependencies
install-newman:
	cd newman && npm install

# Run Go tests
test-go:
	go test ./...

# Format Go code
fmt:
	go fmt ./...

# Tidy Go modules
tidy:
	go mod tidy

# Show service status
status:
	docker-compose ps

# Rebuild and restart Scout service only
rebuild-scout:
	docker-compose build scout
	docker-compose up -d scout
	@echo "Scout service rebuilt and restarted"
