# Single Dockerfile for all three Go binaries.
#
# Usage:
#   docker build --build-arg BINARY=cmd/main.go         -t zpay/api      .
#   docker build --build-arg BINARY=cmd/consumer/main.go -t zpay/consumer .
#   docker build --build-arg BINARY=cmd/producer/main.go -t zpay/producer .
#
# The CI pipeline passes --build-arg BINARY=<path> per job; no other changes needed.

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

ARG BINARY
# Fail fast if the caller forgot to pass BINARY
RUN test -n "$BINARY" || (echo "ERROR: --build-arg BINARY=<path/to/main.go> is required" && exit 1)

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/app \
    ./$BINARY

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/app /app/server

# At runtime k8s mounts:
#   ConfigMap → /app/config/config.yaml
#   Secret    → /app/secret/jwt-secret.txt  (API only)
WORKDIR /app

USER nonroot:nonroot
EXPOSE 8000

ENTRYPOINT ["/app/server"]
