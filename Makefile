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

.PHONY: build clean test lint web-build benchmark coverage race integration e2e docker security

web-build:
	@if [ -f $(WEB_DIR)/package.json ]; then \
		cd $(WEB_DIR) && npm ci && npm run build; \
	fi

build: web-build
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(MAIN)

clean:
	rm -rf $(BIN_DIR)
	rm -rf coverage/

test: web-build
	go test ./...

test-race:
	go test -race ./...

test-v:
	go test -v ./...

benchmark:
	go test -bench=. -benchmem ./test/benchmark/...
	go test -bench=. -benchmem -run=^$$ ./internal/...

coverage:
	@mkdir -p coverage
	go test -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"
	@go tool cover -func=coverage/coverage.out | tail -1

coverage-report: coverage
	@echo "Opening coverage report..."
	@if command -v xdg-open >/dev/null; then xdg-open coverage/coverage.html; \
	elif command -v open >/dev/null; then open coverage/coverage.html; \
	elif command -v start >/dev/null; then start coverage/coverage.html; \
	fi

integration:
	go test -tags=integration ./test/...

e2e:
	go test -tags=e2e ./test/...

lint: web-build
	go vet ./...
	@if command -v golangci-lint >/dev/null; then golangci-lint run; fi

fmt:
	go fmt ./...

fmt-check:
	@if [ -n "$$(go fmt ./...)" ]; then echo "Code is not formatted"; exit 1; fi

deps:
	go mod download
	go mod verify

deps-update:
	go get -u ./...
	go mod tidy

docker:
	docker build -t apicerberus:$(VERSION) .

docker-compose-up:
	docker-compose -f deployments/docker/docker-compose.standalone.yml up -d

docker-compose-down:
	docker-compose -f deployments/docker/docker-compose.standalone.yml down

security:
	@if command -v gosec >/dev/null; then gosec ./...; fi
	@if command -v govulncheck >/dev/null; then govulncheck ./...; fi
	@if command -v trivy >/dev/null; then trivy fs .; fi

changelog:
	@git log --pretty=format:"- %s (%h)" $(shell git describe --tags --abbrev=0 2>/dev/null || echo HEAD~10)..HEAD

all: fmt lint test-race build
