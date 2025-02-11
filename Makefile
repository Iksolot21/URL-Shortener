# Makefile

# General settings
APP_NAME := url-shortener
BINARY_NAME := $(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty)

# Go settings
GO := go
GO_BUILD := $(GO) build
GO_TEST := $(GO) test
GO_MOD := $(GO) mod
GO_LINT := golangci-lint run

# Docker settings
DOCKER := docker
DOCKER_BUILD := $(DOCKER) build
DOCKER_RUN := $(DOCKER) run
DOCKER_IMAGE_NAME := url-shortener

# Directories
CMD_DIR := ./cmd/$(APP_NAME)
CONFIG_DIR := ./config
INTERNAL_DIR := ./internal
TESTS_DIR := ./tests

# Build flags
BUILD_FLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@$(GO_BUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) $(CMD_DIR)

# Run tests
test:
	@echo "Running tests..."
	@$(GO_TEST) -v ./...

# Lint the code
lint:
	@echo "Running linter..."
	@golangci-lint run

# Docker build
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE_NAME)..."
	@$(DOCKER_BUILD) -t $(DOCKER_IMAGE_NAME) .

# Docker run
docker-run: docker-build
	@echo "Running Docker container $(DOCKER_IMAGE_NAME)..."
	@$(DOCKER_RUN) -p 8082:8082 -e STORAGE_TYPE=memory $(DOCKER_IMAGE_NAME)

# Clean the project
clean:
	@echo "Cleaning the project..."
	@$(GO) clean
	@rm -f $(BINARY_NAME)

# Update dependencies
mod-tidy:
	@echo "Tidying Go modules..."
	@$(GO_MOD) tidy

# Help message
help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make test           - Run unit tests"
	@echo "  make lint           - Run linter"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Build and run Docker container"
	@echo "  make clean          - Clean the project"
	@echo "  make mod-tidy       - Tidy Go modules"
	@echo "  make help           - Show this help message"

.PHONY: all build test lint docker-build docker-run clean mod-tidy help