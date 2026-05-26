# Woodstar Makefile

ifneq (,$(wildcard .env))
    include .env
    export
endif

BINARY_NAME = woodstar
WEB_DIR = web

OPENAPI_FILE = $(WEB_DIR)/openapi.yaml

.PHONY: all build backend frontend dev dev-backend dev-frontend test test-integration test-openapi openapi openapi-types lint fmt precommit clean deps schema-sync

all: build

build: frontend backend

backend:
	go build -o $(BINARY_NAME) ./cmd/woodstar

frontend:
	cd $(WEB_DIR) && pnpm install && pnpm build

dev:
	$(MAKE) -j 2 dev-backend dev-frontend

dev-backend:
	air -c .air.toml

dev-frontend:
	cd $(WEB_DIR) && pnpm dev

test:
	go test -race -count=1 -v ./...

test-integration:
	go test -race -count=1 -v ./internal/agentauth ./internal/users ./internal/orbit/protocol ./internal/osquery/protocol ./internal/santa/protocol

openapi:
	go run ./cmd/woodstar openapi --output $(OPENAPI_FILE)

openapi-types: openapi
	cd $(WEB_DIR) && pnpm openapi:types

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

lint:
	golangci-lint run --timeout=5m

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './web/*')

precommit: fmt test

clean:
	rm -rf $(BINARY_NAME) build web/dist internal/web/dist

deps:
	go mod download
	cd $(WEB_DIR) && pnpm install

schema-sync:
	./schema/sync.sh
