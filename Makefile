BINARY := apicerberus
BIN_DIR := bin
MAIN := ./cmd/apicerberus
WEB_DIR := web

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X github.com/APICerberus/APICerebrus/internal/version.Version=$(VERSION) \
	-X github.com/APICerberus/APICerebrus/internal/version.Commit=$(COMMIT) \
	-X github.com/APICerberus/APICerebrus/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: build clean test lint web-build

web-build:
	@if [ -f $(WEB_DIR)/package.json ]; then \
		cd $(WEB_DIR) && npm ci && npm run build; \
	fi

build: web-build
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(MAIN)

clean:
	rm -rf $(BIN_DIR)

test: web-build
	go test ./...

lint: web-build
	go vet ./...
