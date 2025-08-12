.PHONY: test test-short test-state test-providers test-cmd build clean help

TIMEOUT ?= 20m

test:
	go test -v -timeout $(TIMEOUT) ./...

test-short:
	go test -v -short -timeout 5m ./...

test-state:
	go test -v -timeout $(TIMEOUT) ./pkg/state

test-providers:
	go test -v -timeout $(TIMEOUT) ./pkg/providers

test-cmd:
	go test -v -timeout $(TIMEOUT) ./cmd

test-integration:
	go test -v -timeout $(TIMEOUT) -run Integration ./...

test-with-state:
	go test -v -timeout $(TIMEOUT) -run StateManager ./...

build:
	go build -o atlas-cli

build-race:
	go build -race -o atlas-cli

clean:
	rm -f atlas-cli
	rm -f *.db
	rm -f test-*.yaml
	find . -name "*.db" -delete
	find . -name "*test*" -type d -exec rm -rf {} + 2>/dev/null || true

fmt:
	go fmt ./...

lint:
	golangci-lint run

vet:
	go vet ./...

clean-db:
	rm -f *.db
	find . -name "*.db" -delete

reset-db: clean-db
	rm -f state.db

coverage:
	go test -v -timeout $(TIMEOUT) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

bench:
	go test -v -timeout $(TIMEOUT) -bench=. -benchmem ./...

ci-test:
	CI=true go test -v -timeout $(TIMEOUT) -short ./...

help:
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)