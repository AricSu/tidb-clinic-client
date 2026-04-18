GO ?= go
CARGO ?= cargo
PYTHON ?= python3
BIN_DIR ?= bin
CLI_NAME ?= clinic-client
CLI_PKG := ./cmd/clinic-client
COMPILER_DIR := ./compiler-rs
COMPILER_MANIFEST := $(COMPILER_DIR)/Cargo.toml
PORT ?= 8765

.DEFAULT_GOAL := build

.PHONY: help build fmt test vet check wasm viewer sync-cases clean

help:
	@printf "Available targets:\n"
	@printf "  make            Build $(BIN_DIR)/$(CLI_NAME)\n"
	@printf "  make build      Build $(BIN_DIR)/$(CLI_NAME)\n"
	@printf "  make fmt        Format Go code\n"
	@printf "  make test       Run Go tests and compiler-rs cargo tests\n"
	@printf "  make vet        Run go vet\n"
	@printf "  make check      Run fmt, test, vet, and build\n"
	@printf "  make wasm       Rebuild compiler-rs WASM asset\n"
	@printf "  make viewer     Start compiler-rs regression viewer\n"
	@printf "  make sync-cases Regenerate compiler-rs regression cases\n"
	@printf "  make clean      Remove local build artifacts\n"

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(CLI_NAME) $(CLI_PKG)

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...
	$(CARGO) test --manifest-path $(COMPILER_MANIFEST) --workspace

vet:
	$(GO) vet ./...

check: fmt test vet build

wasm:
	./compiler-rs/bindings/go/build-wasm.sh

viewer:
	$(CARGO) build --manifest-path $(COMPILER_MANIFEST)
	$(PYTHON) ./compiler-rs/tools/regression_viewer_py/server.py --compiler-bin ./compiler-rs/target/debug/compiler-rs --port $(PORT)

sync-cases:
	$(CARGO) build --manifest-path $(COMPILER_MANIFEST)
	$(PYTHON) ./compiler-rs/tools/sync_compiler_cases.py --compiler-bin ./compiler-rs/target/debug/compiler-rs

clean:
	rm -rf $(BIN_DIR) $(COMPILER_DIR)/target
