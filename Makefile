VERSION ?= $(shell git describe --tags --always --dirty --first-parent 2>/dev/null || echo "dev")

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/sro .

.PHONY: fmt
fmt:
	go fmt ./...
	go mod tidy

.PHONY: lint
lint:
	go fix ./...
	go vet ./...
	golangci-lint run

.PHONY: test
test:
	go test ./... -v

.PHONY: all
all: fmt lint test build
