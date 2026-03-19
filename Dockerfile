# --- Build Fronted ---
FROM oven/bun:alpine AS fe-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock  ./
RUN bun install
COPY frontend/ .
RUN bun run build

# --- Build Backend ---
# FROM github.com/zenkiet/zenreply:lastest AS be-builder
FROM golang:tip-alpine AS be-builder
WORKDIR /app/backend
# RUN apk add --no-cache git build-base

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
COPY --from=fe-builder /app/frontend/dist/frontend/browser ./cmd/api/static/dist

RUN CGO_ENABLED=0 GOOS=linux go build -o zenreply ./cmd/api/main.go

# --- Build Image ---
FROM alpine:latest AS runner
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root
COPY --from=be-builder /app/backend/zenreply .

EXPOSE 8080

CMD ["./zenreply"]