.PHONY: all build clean run test install deps fmt vet lint

# Variables
BINARY_NAME=strix
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/strix/main.go
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w -X main.Version=$$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')"

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@$(GO) clean
	@echo "Clean complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

# Run vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "Vet complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...
	@echo "Tests complete"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p bin

	@echo "Building for Linux amd64..."
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

	@echo "Building for Linux arm64..."
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

	@echo "Building for Darwin amd64..."
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)

	@echo "Building for Darwin arm64..."
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

	@echo "Building for Windows amd64..."
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

	@echo "Multi-platform build complete"

# Install the binary to GOPATH
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(GOFLAGS) $(LDFLAGS) $(MAIN_PATH)
	@echo "Installation complete"

# Development mode with live reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/air-verse/air@latest"; \
		echo "Running without live reload..."; \
		$(MAKE) run; \
	fi

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t strix:latest .
	@echo "Docker image built: strix:latest"

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -v $(PWD)/data:/data strix:latest

# Check code quality
check: fmt vet lint test
	@echo "Code quality check complete"

# Help
help:
	@echo "Strix - Smart IP Camera Stream Discovery System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          - Build the application"
	@echo "  make run           - Build and run the application"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make deps          - Install dependencies"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run linter"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make build-all     - Build for multiple platforms"
	@echo "  make install       - Install to GOPATH"
	@echo "  make dev           - Run in development mode with live reload"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"
	@echo "  make check         - Run all quality checks"
	@echo "  make help          - Show this help message"