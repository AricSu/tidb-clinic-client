.PHONY: help test wasm viewer sync-cases

ROOT_DIR := ..

help:
	@printf "Available targets:\n"
	@printf "  make test        Run compiler-rs cargo tests through the repo root\n"
	@printf "  make wasm        Rebuild compiler-rs WASM asset\n"
	@printf "  make viewer      Start compiler-rs regression viewer\n"
	@printf "  make sync-cases  Regenerate compiler-rs regression cases\n"

test:
	$(MAKE) -C $(ROOT_DIR) test

wasm:
	$(MAKE) -C $(ROOT_DIR) wasm

viewer:
	$(MAKE) -C $(ROOT_DIR) viewer

sync-cases:
	$(MAKE) -C $(ROOT_DIR) sync-cases
