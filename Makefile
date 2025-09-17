# Viper Build System
# Production-ready Makefile for building, testing, and deploying Viper components

.PHONY: all build test clean install deps lint format check-format security-scan help rootfs-build rootfs-build-gpu rootfs-validate rootfs-release rootfs-clean rootfs-info
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := viper
AGENT_BINARY_NAME := viper-agent
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go configuration
GO := go
GOFMT := gofmt
GOLINT := golangci-lint
GO_VERSION := $(shell $(GO) version | cut -d' ' -f3)

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -w -s"
BUILD_FLAGS := -trimpath $(LDFLAGS)

# Directories
BIN_DIR := bin
DIST_DIR := dist
COVERAGE_DIR := coverage
ROOTFS_DIR := rootfs

# Platform targets for cross-compilation
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64

help: ## Show this help message
	@echo "Viper Build System - Production-Grade Browser Automation Framework"
	@echo "=================================================================="
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Build Info:"
	@echo "  Version:    $(VERSION)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Git Commit: $(GIT_COMMIT)"

deps: ## Install build dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod verify
	@if ! command -v $(GOLINT) >/dev/null 2>&1; then \
		echo "golangci-lint not found - install manually if needed for linting"; \
	fi

format: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) -w -s .
	$(GO) mod tidy

check-format: ## Check if code is formatted
	@echo "Checking code format..."
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "Code is not formatted. Run 'make format'"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi

lint: ## Run linter
	@echo "Running linter..."
	$(GOLINT) run --timeout=5m ./...

security-scan: ## Run security scanner
	@echo "Running security scan..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		$(GO) install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	gosec ./...

test: ## Run tests
	@echo "Running tests..."
	mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	$(GO) test -v -race ./internal/... ./pkg/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GO) test -v -race -tags=integration ./tests/integration/...

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -v -bench=. -benchmem ./...

build: deps ## Build CLI and Agent binaries
	@echo "Building binaries..."
	mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/viper
	$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(AGENT_BINARY_NAME) ./cmd/agent
	@echo "Binaries built:"
	@ls -la $(BIN_DIR)/

build-cli: deps ## Build CLI binary only
	@echo "Building CLI binary..."
	mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/viper

build-agent: deps ## Build Agent binary only
	@echo "Building Agent binary..."
	mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(AGENT_BINARY_NAME) ./cmd/agent

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		echo "Building for $$OS/$$ARCH..."; \
		GOOS=$$OS GOARCH=$$ARCH $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$$OS-$$ARCH ./cmd/viper; \
		GOOS=$$OS GOARCH=$$ARCH $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(AGENT_BINARY_NAME)-$$OS-$$ARCH ./cmd/agent; \
	done
	@echo "Cross-platform binaries built:"
	@ls -la $(DIST_DIR)/

install: build ## Install binaries to system PATH
	@echo "Installing binaries..."
	cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	cp $(BIN_DIR)/$(AGENT_BINARY_NAME) $(GOPATH)/bin/
	@echo "Binaries installed to $(GOPATH)/bin/"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	$(GO) clean -cache -testcache -modcache

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	docker build -f rootfs/Dockerfile -t viper-rootfs:$(VERSION) .
	docker build -f docker/Dockerfile.agent -t viper-agent:$(VERSION) .

quality: check-format lint security-scan test ## Run all quality checks

ci: deps quality build ## Run CI pipeline (deps, quality, build)

release: clean quality build-all ## Prepare release artifacts
	@echo "Creating release artifacts..."
	mkdir -p $(DIST_DIR)/checksums
	@for file in $(DIST_DIR)/viper-* $(DIST_DIR)/viper-agent-*; do \
		if [ -f "$$file" ]; then \
			sha256sum "$$file" > "$$file.sha256"; \
			echo "Generated checksum for $$(basename $$file)"; \
		fi \
	done
	@echo "Release artifacts ready in $(DIST_DIR)/"

dev: build ## Quick development build
	@echo "Development build complete"
	@$(BIN_DIR)/$(BINARY_NAME) --version

# Development helpers
watch-test: ## Watch for changes and run tests
	@if ! command -v fswatch >/dev/null 2>&1; then \
		echo "fswatch not installed. Install with: brew install fswatch"; \
		exit 1; \
	fi
	fswatch -o . -e ".*" -i "\.go$$" | xargs -n1 -I{} make test-unit

nomad-job: ## Generate Nomad job files
	@echo "Nomad job files available in jobs/ directory"
	@ls -la jobs/*.hcl

.PHONY: version
rootfs-build: build-agent ## Build rootfs image using Packer
	@echo "Building rootfs image with Packer..."
	@if ! command -v packer >/dev/null 2>&1; then \
		echo "ERROR: Packer not found. Install from: https://www.packer.io/downloads"; \
		exit 1; \
	fi
	@if [ ! -f "$(BIN_DIR)/$(AGENT_BINARY_NAME)" ]; then \
		echo "ERROR: Agent binary not found. Run 'make build-agent' first"; \
		exit 1; \
	fi
	@echo "Building rootfs image version: $(VERSION)"
	cd $(ROOTFS_DIR) && packer build \
		-var "version=$(VERSION)" \
		-var "output_dir=../$(DIST_DIR)/rootfs" \
		alpine.pkr.hcl
	@echo "Rootfs image built successfully!"
	@echo "Output: $(DIST_DIR)/rootfs/viper-rootfs-$(VERSION)-*/viper-rootfs-$(VERSION)-*.qcow2"

rootfs-build-gpu: build-agent ## Build rootfs image with GPU support
	@echo "Building GPU-enabled rootfs image with Packer..."
	@if ! command -v packer >/dev/null 2>&1; then \
		echo "ERROR: Packer not found. Install from: https://www.packer.io/downloads"; \
		exit 1; \
	fi
	@if [ ! -f "$(BIN_DIR)/$(AGENT_BINARY_NAME)" ]; then \
		echo "ERROR: Agent binary not found. Run 'make build-agent' first"; \
		exit 1; \
	fi
	@echo "Building GPU-enabled rootfs image version: $(VERSION)"
	cd $(ROOTFS_DIR) && packer build \
		-var "version=$(VERSION)-gpu" \
		-var "output_dir=../$(DIST_DIR)/rootfs" \
		-var "enable_gpu=true" \
		alpine.pkr.hcl
	@echo "GPU-enabled rootfs image built successfully!"
	@echo "Output: $(DIST_DIR)/rootfs/viper-rootfs-$(VERSION)-gpu-*/viper-rootfs-$(VERSION)-gpu-*.qcow2"

rootfs-validate: ## Validate Packer template
	@echo "Validating Packer template..."
	@if ! command -v packer >/dev/null 2>&1; then \
		echo "ERROR: Packer not found. Install from: https://www.packer.io/downloads"; \
		exit 1; \
	fi
	cd $(ROOTFS_DIR) && packer validate alpine.pkr.hcl
	@echo "Packer template validation passed!"

rootfs-release: rootfs-build ## Build and prepare rootfs for release
	@echo "Preparing rootfs release artifacts..."
	mkdir -p $(DIST_DIR)/rootfs/release
	@echo "Copying latest rootfs images to release directory..."
	@find $(DIST_DIR)/rootfs -name "*.qcow2" -type f -exec cp {} $(DIST_DIR)/rootfs/release/ \;
	@find $(DIST_DIR)/rootfs -name "metadata.json" -type f -exec cp {} $(DIST_DIR)/rootfs/release/ \;
	@echo "Generating checksums for rootfs images..."
	@cd $(DIST_DIR)/rootfs/release && for file in *.qcow2; do \
		if [ -f "$$file" ]; then \
			sha256sum "$$file" > "$$file.sha256"; \
			echo "Generated checksum for $$file"; \
		fi \
	done
	@echo "Rootfs release artifacts ready in $(DIST_DIR)/rootfs/release/"
	@ls -la $(DIST_DIR)/rootfs/release/

rootfs-clean: ## Clean rootfs build artifacts
	@echo "Cleaning rootfs artifacts..."
	rm -rf $(DIST_DIR)/rootfs
	rm -rf $(ROOTFS_DIR)/out
	@echo "Rootfs artifacts cleaned."

rootfs-info: ## Show information about built rootfs images
	@echo "Rootfs Images Information:"
	@echo "========================="
	@if [ -d "$(DIST_DIR)/rootfs" ]; then \
		find $(DIST_DIR)/rootfs -name "metadata.json" -exec echo "--- {} ---" \; -exec cat {} \; -exec echo "" \; 2>/dev/null || echo "No metadata files found"; \
		echo ""; \
		echo "Image Files:"; \
		find $(DIST_DIR)/rootfs -name "*.qcow2" -exec ls -lh {} \; 2>/dev/null || echo "No qcow2 files found"; \
	else \
		echo "No rootfs images found. Run 'make rootfs-build' first."; \
	fi

.PHONY: version
version: ## Show version information
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(GO_VERSION)"