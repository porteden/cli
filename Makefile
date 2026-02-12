.PHONY: build install test clean fmt lint run

# Variables
BINARY_NAME=porteden
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT?=$(shell git rev-parse --short=12 HEAD 2>/dev/null || echo "")
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X github.com/porteden/cli/internal/config.Version=$(VERSION) -X github.com/porteden/cli/internal/config.Commit=$(COMMIT) -X github.com/porteden/cli/internal/config.Date=$(DATE)"

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/porteden

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/porteden

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f porteden-*
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Run the CLI
run:
	go run ./cmd/porteden

# Download dependencies
deps:
	go mod download
	go mod tidy

# Cross-compile for all platforms
build-all:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o porteden-darwin-arm64 ./cmd/porteden
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o porteden-darwin-amd64 ./cmd/porteden
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o porteden-linux-amd64 ./cmd/porteden
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o porteden-windows-amd64.exe ./cmd/porteden

# Development build with race detector
dev:
	go build -race $(LDFLAGS) -o $(BINARY_NAME) ./cmd/porteden

# Show help
help:
	@echo "PortEden CLI - Makefile Commands"
	@echo ""
	@echo "  make build       - Build the binary"
	@echo "  make install     - Install to GOPATH/bin"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make run         - Run the CLI"
	@echo "  make deps        - Download dependencies"
	@echo "  make build-all   - Cross-compile for all platforms"
	@echo "  make dev         - Development build with race detector"
	@echo ""
	@echo "Build with version:"
	@echo "  make build VERSION=1.0.0"
