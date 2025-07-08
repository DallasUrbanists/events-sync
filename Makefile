.PHONY: build test clean migrate migrate-version docker-build docker-run

# Build the main application
build:
	go build -o bin/events-sync ./cmd/events-sync

# Build the migration tool
build-migrate:
	go build -o bin/migrate ./cmd/migrate

# Build the web server
build-web:
	go build -o bin/web ./cmd/web

# Run all tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Run migrations
migrate:
	go run ./cmd/migrate -action=up

# Check migration version
migrate-version:
	go run ./cmd/migrate -action=version

# Run web server
web:
	go run ./cmd/web

# Run web server with custom port
web-port:
	go run ./cmd/web -port=3000

# Build Docker image
docker-build:
	docker build -t events-sync .

# Run with Docker Compose
docker-run:
	docker-compose up --build

# Run with Docker Compose in background
docker-run-detached:
	docker-compose up -d --build

# Stop Docker Compose
docker-stop:
	docker-compose down

# Show Docker Compose logs
docker-logs:
	docker-compose logs -f

# Create a new migration file
migrate-create:
	@read -p "Enter migration name: " name; \
	timestamp=$$(date +%s); \
	up_file="migrations/$${timestamp}_$${name}.up.sql"; \
	down_file="migrations/$${timestamp}_$${name}.down.sql"; \
	echo "-- Migration: $$name" > "$$up_file"; \
	echo "-- Migration: $$name" > "$$down_file"; \
	echo "Created migration files:"; \
	echo "  $$up_file"; \
	echo "  $$down_file"

# Help
help:
	@echo "Available commands:"
	@echo "  build              - Build the main application"
	@echo "  build-migrate      - Build the migration tool"
	@echo "  test               - Run all tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  clean              - Clean build artifacts"
	@echo "  migrate            - Run database migrations"
	@echo "  migrate-version    - Check current migration version"
	@echo "  migrate-create     - Create a new migration file"
	@echo "  web                - Run web server"
	@echo "  web-port           - Run web server on port 3000"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run with Docker Compose"
	@echo "  docker-run-detached - Run with Docker Compose in background"
	@echo "  docker-stop        - Stop Docker Compose"
	@echo "  docker-logs        - Show Docker Compose logs"
	@echo "  help               - Show this help message"