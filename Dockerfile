# ============================================================
# ZenReply — Multi-stage Dockerfile
# Stage 1: Build Frontend (Angular / Bun)
# Stage 2: Build Backend (Golang)
# Stage 3: Minimal production image (Alpine)
# ============================================================

# ── Stage 1: Frontend Build ──────────────────────────────────
FROM oven/bun:alpine AS fe-builder
WORKDIR /app/frontend

COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile

COPY frontend/ .
RUN bun run build

# ── Stage 2: Backend Build ────────────────────────────────────
FROM golang:1.22-alpine AS be-builder
WORKDIR /app/backend

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Download dependencies first (layer cache optimization)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY backend/ .

# Copy built frontend assets into the backend static directory
COPY --from=fe-builder /app/frontend/dist/frontend/browser ./cmd/api/static/dist

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o zenreply ./cmd/api/main.go

# ── Stage 3: Production Image ─────────────────────────────────
FROM alpine:3.20 AS runner

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1001 -S zenreply && \
    adduser -u 1001 -S zenreply -G zenreply

WORKDIR /app

# Copy binary
COPY --from=be-builder --chown=zenreply:zenreply /app/backend/zenreply .

# Switch to non-root user
USER zenreply

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./zenreply"]
