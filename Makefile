SHELL := /bin/sh

BACKEND_DIR := apps/pocket-music/backend
FRONTEND_DIR := apps/pocket-music/frontend
MUSICD_ADDR ?= :8765
VITE_HOST ?= 0.0.0.0
BUILD_DIR ?= build

.PHONY: dev backend frontend test test-backend test-frontend test-unit build serve bench screenshots

# Start musicd and Vite together; Ctrl-C stops both processes.
dev:
	@set -eu; \
	  (cd $(BACKEND_DIR) && MUSICD_ADDR='$(MUSICD_ADDR)' go run .) & backend_pid=$$!; \
	  frontend_pid=; \
	  cleanup() { \
	    trap - EXIT INT TERM; \
	    if [ -n "$$frontend_pid" ]; then kill "$$frontend_pid" 2>/dev/null || true; fi; \
	    kill "$$backend_pid" 2>/dev/null || true; \
	    wait "$$frontend_pid" 2>/dev/null || true; \
	    wait "$$backend_pid" 2>/dev/null || true; \
	  }; \
	  trap cleanup EXIT INT TERM; \
	  npm --prefix $(FRONTEND_DIR) run dev -- --host $(VITE_HOST) & frontend_pid=$$!; \
	  wait "$$frontend_pid"

backend:
	cd $(BACKEND_DIR) && MUSICD_ADDR='$(MUSICD_ADDR)' go run .

frontend:
	npm --prefix $(FRONTEND_DIR) run dev -- --host $(VITE_HOST)

# Backend tests, frontend types, pure-logic unit tests, i18n catalog parity.
test: test-backend test-frontend

test-backend:
	cd $(BACKEND_DIR) && go test ./...

test-unit:
	npm --prefix $(FRONTEND_DIR) run test:unit

test-frontend:
	npm --prefix $(FRONTEND_DIR) run typecheck
	npm --prefix $(FRONTEND_DIR) run test:unit
	@tmp=$$(mktemp); $(FRONTEND_DIR)/node_modules/.bin/esbuild $(FRONTEND_DIR)/src/i18n-contract.ts --bundle --platform=node --outfile=$$tmp --log-level=warning && node $$tmp; rm -f $$tmp

# Single-binary desktop build: musicd serves the compiled UI from MUSICD_WEB_DIR.
build:
	cd $(BACKEND_DIR) && go build -o ../../$(BUILD_DIR)/musicd .
	npm --prefix $(FRONTEND_DIR) run build

serve: build
	MUSICD_WEB_DIR=$(abspath $(FRONTEND_DIR)/dist) MUSICD_MOCK=true ./$(BUILD_DIR)/musicd

# Library-scale benchmarks (10k tracks) for search, paging and the album grid.
bench:
	cd $(BACKEND_DIR) && go test -run '^$$' -bench '10K' -benchtime=10x -benchmem ./...

# Requires a running instance on 127.0.0.1:8765 (`make serve` or `make dev`).
screenshots:
	node apps/pocket-music/scripts/capture-screenshots.mjs --base http://127.0.0.1:8765
