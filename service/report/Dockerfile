# syntax=docker/dockerfile:1
# Build context = root repo (xem deployments/docker/docker-compose.yml)
# Service này là 1 Go module riêng, dùng module `taskmanager/common` qua replace directive.
# Image chứa 2 binary: grpc (gRPC server) + http (grpc-gateway).
# Report service không có DB riêng: tổng hợp dữ liệu qua task service (TASK_GRPC_DIAL_ADDR).

# ---------------------------------------------------------------- builder
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Layer cache: chỉ copy go.mod/go.sum trước
COPY common/go.mod common/go.sum ./common/
COPY service/report/go.mod service/report/go.sum ./service/report/
RUN cd service/report && go mod download

# Source
COPY common/ ./common/
COPY service/report/ ./service/report/

RUN cd service/report \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/grpc ./cmd/grpc \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/http ./cmd/http

# ---------------------------------------------------------------- runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 app

COPY --from=builder /out/grpc /usr/local/bin/grpc
COPY --from=builder /out/http /usr/local/bin/http

USER app
EXPOSE 8087 9087

# Mặc định chạy gRPC server; container gateway override entrypoint (xem docker-compose.yml).
ENTRYPOINT ["/usr/local/bin/grpc"]
