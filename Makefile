.PHONY: build test clean run-stdio run-sse run-streamable-http

# Binary name
BINARY_NAME=luno-mcp

# Build the application
build:
	go build -o $(BINARY_NAME) ./cmd/server

# Run all tests
test:
	go test ./...

# Clean build files
clean:
	go clean
	rm -f $(BINARY_NAME)

# Run in stdio mode
run-stdio:
	go run ./cmd/server

# Run in SSE mode
run-sse:
	go run ./cmd/server --transport sse --sse-address localhost:8080

# Run in Streamable HTTP mode
run-streamable-http:
	go run ./cmd/server --transport streamable-http --sse-address localhost:8080

# Install the binary to your GOBIN path
install:
	go install ./cmd/server

pre-commit:
	pre-commit install

# Default target
default: build
