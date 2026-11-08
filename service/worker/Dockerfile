# syntax=docker/dockerfile:1
# Build context = root repo (xem deployments/docker/docker-compose.yml)
# worker-service: Kafka consumer, KHÔNG có gRPC/HTTP API (chỉ /healthz).
# Scale ngang bằng số replica: `docker compose up -d --scale worker=3`.

# ---------------------------------------------------------------- builder
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY common/go.mod common/go.sum ./common/
COPY service/worker/go.mod service/worker/go.sum ./service/worker/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd service/worker && go mod download

COPY common/ ./common/
COPY service/worker/ ./service/worker/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd service/worker \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---------------------------------------------------------------- runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 app

COPY --from=builder /out/worker /usr/local/bin/worker

USER app
EXPOSE 8094

ENTRYPOINT ["/usr/local/bin/worker"]
