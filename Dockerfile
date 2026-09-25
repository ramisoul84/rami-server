# ============================================
# BUILD STAGE
# ============================================
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependencies separately from source
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .

# Static binary — no libc dependency, runs on scratch/alpine
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.version=$(date +%Y%m%d-%H%M%S)" \
    -o /build/bin/server \
    ./cmd/server

# ============================================
# RUNTIME STAGE
# ============================================
FROM alpine:3.20

# Runtime essentials
RUN apk add --no-cache ca-certificates tzdata wget

# Non-root user
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/server /app/server

# Copy migrations (applied by ops, not by the app itself)
COPY --from=builder /build/migrations /app/migrations

# Geo database is optional — mount it as a volume, or COPY it here
# COPY ./data/GeoLite2-City.mmdb /app/data/GeoLite2-City.mmdb
RUN mkdir -p /app/data

# Logs directory
RUN mkdir -p /app/logs && chown -R app:app /app

USER app

EXPOSE 8080

# Health check — hits the /health endpoint every 30s
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/server"]