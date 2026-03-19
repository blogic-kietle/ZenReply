GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOPATH=$(shell go env GOPATH)

.PHONY: all build clean run test lint fmt deps

all: build

deps:
	@echo " > Installing dependencies..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest

dev:
	@echo " > Starting development server..."
	go run ./cmd/api/main.go

format:
	@echo " > Formatting code..."
	gofumpt -w .
	goimports -w -local github.com/zenkiet/zenreply .

lint:
	@echo " > Linting..."
	golangci-lint run ./...