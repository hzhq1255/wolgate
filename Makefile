# wolgate Makefile
# Lightweight Wake-on-LAN gateway for routers

BINARY_NAME=wolgate
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Go build flags
GO=go
GOFLAGS=-v
CGO_ENABLED=0

# Directories
SRC_DIR=.
BUILD_DIR=build

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)/main.go

# Build for Linux ARM (路由器常见架构)
.PHONY: build-arm
build-arm:
	@echo "Building $(BINARY_NAME) for linux/arm..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-arm $(SRC_DIR)/main.go

# Build for Linux ARM64
.PHONY: build-arm64
build-arm64:
	@echo "Building $(BINARY_NAME) for linux/arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 $(SRC_DIR)/main.go

# Build for Linux MIPS (OpenWrt 常见架构)
.PHONY: build-mips
build-mips:
	@echo "Building $(BINARY_NAME) for linux/mips..."
	CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-mips $(SRC_DIR)/main.go

# Build for Linux MIPSEL (小端序)
.PHONY: build-mipsle
build-mipsle:
	@echo "Building $(BINARY_NAME) for linux/mipsle..."
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-mipsle $(SRC_DIR)/main.go

# Build for all supported platforms
.PHONY: build-all
build-all: build build-arm build-arm64 build-mips build-mipsle
	@echo "Built $(BINARY_NAME) for all platforms"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run the application
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) server

# Install dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Show binary size
.PHONY: size
size: build
	@echo "Binary sizes:"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)* 2>/dev/null || echo "No binaries found in $(BUILD_DIR)"

# Create build directory
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Development build (with debug info)
.PHONY: dev
dev:
	@echo "Building $(BINARY_NAME) for development..."
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)/main.go

# Help target
.PHONY: help
help:
	@echo "wolgate Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all           Build for current platform (default)"
	@echo "  build         Build for current platform"
	@echo "  build-arm     Build for linux/arm"
	@echo "  build-arm64   Build for linux/arm64"
	@echo "  build-mips    Build for linux/mips"
	@echo "  build-mipsle  Build for linux/mipsle"
	@echo "  build-all     Build for all platforms"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  clean         Clean build artifacts"
	@echo "  run           Build and run server"
	@echo "  deps          Download dependencies"
	@echo "  fmt           Format code"
	@echo "  lint          Run linter"
	@echo "  size          Show binary sizes"
	@echo "  dev           Development build (with debug info)"
	@echo "  help          Show this help message"
