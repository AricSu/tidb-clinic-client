GO ?= go
BIN_DIR ?= bin
CLI_NAME ?= clinic-client
CLI_PKG := ./cmd/clinic-client

.DEFAULT_GOAL := help

.PHONY: help all fmt test vet build build-cli run-cli clean

help:
	@printf "Available targets:\n"
	@printf "  make all        Run fmt, test, vet, and build-cli\n"
	@printf "  make fmt        Format Go code\n"
	@printf "  make test       Run all Go tests\n"
	@printf "  make vet        Run go vet\n"
	@printf "  make build      Build the public CLI\n"
	@printf "  make build-cli  Build the public CLI into $(BIN_DIR)/$(CLI_NAME)\n"
	@printf "  make run-cli    Run clinic-client metrics query-range via go run\n"
	@printf "  make clean      Remove local build artifacts\n"

all: fmt test vet build-cli

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build: build-cli

build-cli:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(CLI_NAME) $(CLI_PKG)

run-cli:
	$(GO) run $(CLI_PKG) metrics query-range

clean:
	rm -rf $(BIN_DIR)
