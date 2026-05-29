# Woodstar Makefile

ifneq (,$(wildcard .env))
    include .env
    export
endif

BINARY_NAME = woodstar
WEB_DIR = web

OPENAPI_FILE = $(WEB_DIR)/openapi.yaml
WOODSTAR_TEST_DATABASE_URL ?= postgres://woodstar:woodstar@127.0.0.1:5432/postgres?sslmode=disable
GO_TEST_FLAGS ?= -race -count=1 -v
GOLANGCI_LINT ?= golangci-lint
PNPM ?= pnpm

.PHONY: all build backend frontend dev dev-backend dev-frontend test full-test test-integration test-openapi openapi openapi-types backend-lint frontend-lint lint backend-format frontend-format format fmt precommit clean deps schema-sync

all: build

build: frontend backend

backend:
	go build -o $(BINARY_NAME) ./cmd/woodstar

frontend:
	cd $(WEB_DIR) && $(PNPM) install && $(PNPM) build

dev:
	$(MAKE) -j 2 dev-backend dev-frontend

dev-backend:
	air -c .air.toml

dev-frontend:
	cd $(WEB_DIR) && $(PNPM) dev

test:
	env -u WOODSTAR_TEST_DATABASE_URL go test $(GO_TEST_FLAGS) ./...

full-test:
	WOODSTAR_TEST_DATABASE_URL="$(WOODSTAR_TEST_DATABASE_URL)" go test $(GO_TEST_FLAGS) ./...

test-integration:
	WOODSTAR_TEST_DATABASE_URL="$(WOODSTAR_TEST_DATABASE_URL)" go test $(GO_TEST_FLAGS) ./internal/agentauth ./internal/users ./internal/orbit/protocol ./internal/osquery/protocol ./internal/santa/protocol

openapi:
	go run ./cmd/woodstar openapi --output $(OPENAPI_FILE)

openapi-types: openapi
	cd $(WEB_DIR) && $(PNPM) openapi:types

test-openapi:
	@tmp=$$(mktemp); \
	go run ./cmd/woodstar openapi --output $$tmp; \
	if ! diff -q $(OPENAPI_FILE) $$tmp >/dev/null; then \
		echo "ERROR: $(OPENAPI_FILE) is out of date. Run 'make openapi-types' and commit the result."; \
		diff -u $(OPENAPI_FILE) $$tmp; \
		rm -f $$tmp; \
		exit 1; \
	fi; \
	rm -f $$tmp

backend-lint:
	$(GOLANGCI_LINT) run
	@deadcode_output="$$(go tool golang.org/x/tools/cmd/deadcode -test ./...)"; \
	if [ -n "$$deadcode_output" ]; then \
		printf '%s\n' "$$deadcode_output"; \
		exit 1; \
	fi

frontend-lint:
	cd $(WEB_DIR) && $(PNPM) run lint

lint: backend-lint frontend-lint

backend-format:
	$(GOLANGCI_LINT) fmt

frontend-format:
	cd $(WEB_DIR) && $(PNPM) run format

format: frontend-format backend-format

fmt: format

precommit: format lint full-test test-openapi

clean:
	rm -rf $(BINARY_NAME) build web/dist internal/web/dist

deps:
	go mod download
	cd $(WEB_DIR) && $(PNPM) install

schema-sync:
	./schema/sync.sh
