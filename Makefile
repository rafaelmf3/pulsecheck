.PHONY: build test clean run

# Build the application
build:
	go build -o bin/pulsecheck ./cmd/node

# Run the application
run: build
	./bin/pulsecheck

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod download
	go mod tidy
