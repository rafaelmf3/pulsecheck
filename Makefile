.PHONY: build test test-unit test-integration clean run deps simulator-up simulator-down simulator-logs simulator-clean

# Build the application
build:
	go build -o bin/pulsecheck ./cmd/node

# Run the application
run: build
	./bin/pulsecheck

# Run all tests
test: test-unit

# Run unit tests
test-unit:
	go test -v -race -coverprofile=coverage.out ./...

# Run unit tests with coverage report
test-coverage: test-unit
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests (requires Docker)
test-integration:
	@echo "Starting integration tests..."
	go test -v ./test/integration/...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Docker Compose simulator commands
simulator-up:
	docker-compose -f docker-compose.yml up -d --build

simulator-down:
	docker-compose -f docker-compose.yml down

simulator-logs:
	docker-compose -f docker-compose.yml logs -f

simulator-clean: simulator-down
	docker-compose -f docker-compose.yml down -v
	docker system prune -f
