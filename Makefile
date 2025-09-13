default: build
.PHONY: build
build: build-migrate build-web build-events-sync-worker

.PHONY: pre-build
pre-build:
	mkdir -p bin

.PHONY: build-migrate
build-migrate: pre-build
	go build -o bin/migrate ./cmd/migrate/main.go

.PHONY: build-web
build-web: pre-build
	go build -o bin/web ./cmd/web/main.go

.PHONY: build-events-sync-worker
build-events-sync-worker: pre-build
	go build -o bin/events-sync ./cmd/events-sync/main.go

.PHONY: test
test:
	go test ./...

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean:
	rm -rf bin/
	rm -f coverage.out

.PHONY: migrate
migrate:
	go run ./cmd/migrate -action=up

.PHONY: migrate-version
migrate-version:
	go run ./cmd/migrate -action=version

.PHONY: migrate-down
migrate-down:
	@if [ -z "$(STEPS)" ]; then \
		echo "Usage: make migrate-down STEPS=N"; \
		echo "Example: make migrate-down STEPS=2"; \
		exit 1; \
	fi
	go run ./cmd/migrate -action=down -steps=$(STEPS)

.PHONY: migrate-create
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
