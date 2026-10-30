SHELL := /bin/sh

BACKEND_DIR := apps/pocket-music/backend
FRONTEND_DIR := apps/pocket-music/frontend
MUSICD_ADDR ?= :8765
VITE_HOST ?= 127.0.0.1

.PHONY: dev backend frontend test test-backend test-frontend

# Start musicd and Vite together; Ctrl-C stops both processes.
dev:
	@set -eu; \
	  MUSICD_ADDR='$(MUSICD_ADDR)' go run ./$(BACKEND_DIR) & backend_pid=$$!; \
	  trap 'kill $$backend_pid 2>/dev/null || true; wait $$backend_pid 2>/dev/null || true' EXIT INT TERM; \
	  npm --prefix $(FRONTEND_DIR) run dev -- --host $(VITE_HOST)

backend:
	MUSICD_ADDR='$(MUSICD_ADDR)' go run ./$(BACKEND_DIR)

frontend:
	npm --prefix $(FRONTEND_DIR) run dev -- --host $(VITE_HOST)

# Run available backend and frontend correctness checks in one command.
test: test-backend test-frontend

test-backend:
	cd $(BACKEND_DIR) && go test ./...

test-frontend:
	npm --prefix $(FRONTEND_DIR) run typecheck
