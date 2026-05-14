# syntax=docker/dockerfile:1

# ---------- build ----------
FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---------- runtime ----------
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/server /app/server
# Migrations are embedded in the binary and applied at startup.
# GeoLite2 database and logs are mounted as volumes (see docker-compose.yml).
RUN mkdir -p /app/data /app/logs && chown -R app:app /app

USER app
EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8000/health >/dev/null || exit 1

ENTRYPOINT ["/app/server"]
