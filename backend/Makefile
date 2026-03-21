GOBASE  := $(shell pwd)
GOBIN   := $(GOBASE)/bin
GOPATH  := $(shell go env GOPATH)
BINARY  := zenreply

.PHONY: all build clean run dev lint fmt vet deps docs docker-build docker-up docker-down docker-logs

all: build

# ── Dependencies ─────────────────────────────────────────────
deps:
	@echo " > Installing dev tools..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo " > Done."

# ── Build ─────────────────────────────────────────────────────
build:
	@echo " > Building $(BINARY)..."
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(GOBIN)/$(BINARY) ./cmd/api/main.go
	@echo " > Binary: $(GOBIN)/$(BINARY)"

# ── Run ───────────────────────────────────────────────────────
dev:
	@echo " > Starting development server..."
	go run ./cmd/api/main.go

run: build
	@echo " > Running $(BINARY)..."
	$(GOBIN)/$(BINARY)

# ── Docs ──────────────────────────────────────────────────────
docs:
	@echo " > Generating Swagger/OpenAPI docs..."
	$(GOPATH)/bin/swag init -g cmd/api/main.go --output docs --parseDependency --parseInternal
	@echo " > Docs generated at docs/"

# ── Code Quality ──────────────────────────────────────────────
fmt:
	@echo " > Formatting code..."
	gofumpt -w .
	goimports -w -local github.com/kietle/zenreply .

lint:
	@echo " > Linting..."
	golangci-lint run ./...

vet:
	@echo " > Running go vet..."
	go vet ./...

# ── Docker ────────────────────────────────────────────────────
docker-build:
	@echo " > Building Docker image..."
	cd .. && docker compose build

docker-up:
	@echo " > Starting services..."
	cd .. && docker compose up -d

docker-down:
	@echo " > Stopping services..."
	cd .. && docker compose down

docker-logs:
	cd .. && docker compose logs -f app

# ── Clean ─────────────────────────────────────────────────────
clean:
	@echo " > Cleaning..."
	rm -rf $(GOBIN)
	go clean -cache
