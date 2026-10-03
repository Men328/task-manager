SHELL := /bin/bash

# Mỗi thư mục trong service/ (và common/) là 1 Go module riêng, quản lý bằng go.work.
# Vì root không phải module nên `go build ./...` không dùng được -> loop từng module.
MODULES := common service/calendar service/identity service/mail-provider service/task tools/quality

GOBIN ?= $(shell go env GOPATH 2>/dev/null)/bin
export PATH := $(GOBIN):$(PATH)

COMPOSE := docker compose -f deployments/docker/docker-compose.yml
# Kèm profile `tunnel` (cloudflared) cho ps/logs/down để không bỏ sót container.
COMPOSE_ALL := $(COMPOSE) --profile tunnel
DATABASE_URL ?= postgres://task_manager:task_manager@postgres:5432/task_manager?sslmode=disable

.DEFAULT_GOAL := help

.PHONY: help tools gen error-codes check-error-codes tidy work build test fmt vet smoke seed-demo migrate-up migrate-down \
        up up-core up-tunnel tunnel-up tunnel-stop down logs tunnel-logs ps build-images web-install web-dev web-build \
        run-identity run-identity-grpc run-identity-http run-task run-task-grpc run-task-http \
        run-calendar run-calendar-grpc run-calendar-http \
        run-mail-provider run-mail-provider-grpc run-mail-provider-http \
        quality-all quality-fmt quality-vet quality-arch

help: ## Hiển thị danh sách lệnh
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ----------------------------------------------------------------- toolchain
tools: ## Cài buf + các protoc plugin
	bash scripts/install-tools.sh

gen: ## Sinh code từ common/proto vào common/gen
	bash scripts/gen.sh

error-codes: ## Sinh bản mirror bộ mã lỗi cho frontend từ common/errorcode
	node scripts/sync-error-codes.mjs

check-error-codes: ## Fail nếu mirror bộ mã lỗi của FE lệch bản canonical
	node scripts/sync-error-codes.mjs --check

tidy: ## go mod tidy cho từng module
	@for m in $(MODULES); do echo "==> $$m"; (cd $$m && go mod tidy) || exit 1; done

work: ## Đồng bộ go.work theo các module
	go work sync

# --------------------------------------------------------------------- build
build: ## Build toàn bộ module
	@for m in $(MODULES); do echo "==> $$m"; (cd $$m && go build ./...) || exit 1; done

test: ## Chạy test Go
	@for m in $(MODULES); do echo "==> $$m"; (cd $$m && go test ./...) || exit 1; done

vet: ## go vet (bỏ qua cảnh báo trong source của dependency)
	bash scripts/vet.sh

fmt: ## gofmt
	gofmt -l -w common service tools

# ------------------------------------------------------------------- quality
quality-all: quality-fmt quality-vet quality-arch check-error-codes ## Kiểm tra chất lượng code (fmt + vet + rules kiến trúc + error codes)

quality-fmt: ## gofmt: fail nếu file chưa format
	@out="$$(gofmt -l common service tools)"; \
	if [ -n "$$out" ]; then echo "gofmt cần sửa:"; echo "$$out"; exit 1; fi; \
	echo "quality-fmt OK"

quality-vet: ## go vet từng module (lọc cảnh báo trong dependency)
	@bash scripts/vet.sh

quality-arch: ## Kiểm tra rules kiến trúc khai báo trong quality.json
	@cd tools/quality && go run . -config ../../quality.json

smoke: ## Smoke test end-to-end (build + start service + gọi API thật)
	bash scripts/smoke-test.sh

seed-demo: ## Seed dữ liệu demo giống design/ui.png (service phải đang chạy)
	node scripts/seed-demo.mjs

# ---------------------------------------------------------------- migrations
migrate-up: ## Chạy toàn bộ migration (qua docker compose)
	$(COMPOSE) run --rm migrate

migrate-down: ## Rollback 1 migration
	$(COMPOSE) run --rm migrate -path=/migrations -database "$(DATABASE_URL)" down 1

# ------------------------------------------------------------------- docker
up: ## docker compose up --build (postgres + migrate + *-grpc + gateway + frontend)
	$(COMPOSE) up --build -d

up-core: ## Chạy stack KHÔNG kèm cloudflared (tắt tunnel, bỏ qua COMPOSE_PROFILES)
	COMPOSE_PROFILES= $(COMPOSE) up --build -d

up-tunnel: ## Như `up` nhưng kèm cloudflared (cần CLOUDFLARE_TUNNEL_TOKEN trong .env)
	$(COMPOSE_ALL) up --build -d

tunnel-up: ## Chỉ bật cloudflared (stack đang chạy sẵn)
	$(COMPOSE_ALL) up -d cloudflared

tunnel-stop: ## Dừng cloudflared, giữ các service khác chạy
	$(COMPOSE_ALL) stop cloudflared

down: ## docker compose down (kèm cloudflared nếu đang chạy)
	$(COMPOSE_ALL) down

down-all: ## docker compose down (kèm cloudflared nếu đang chạy)
	$(COMPOSE_ALL) down -v

logs: ## Xem log compose
	$(COMPOSE) logs -f

tunnel-logs: ## Xem log cloudflared
	$(COMPOSE_ALL) logs -f cloudflared

ps: ## Trạng thái compose (kèm cloudflared nếu đang chạy)
	$(COMPOSE_ALL) ps

build-images: ## Chỉ build image
	$(COMPOSE) build

# --------------------------------------------------------------------- local
# Mỗi service gồm 2 binary: cmd/grpc (gRPC server) + cmd/http (grpc-gateway).
run-identity: ## Chạy identity local: gRPC :9081 + gateway :8081
	bash scripts/run-service.sh identity

run-identity-grpc: ## Chỉ chạy gRPC server identity (:9081)
	cd service/identity && go run ./cmd/grpc

run-identity-http: ## Chỉ chạy HTTP gateway identity (:8081)
	cd service/identity && go run ./cmd/http

run-task: ## Chạy task local: gRPC :9082 + gateway :8082
	bash scripts/run-service.sh task

run-task-grpc: ## Chỉ chạy gRPC server task (:9082)
	cd service/task && go run ./cmd/grpc

run-task-http: ## Chỉ chạy HTTP gateway task (:8082)
	cd service/task && go run ./cmd/http

run-calendar: ## Chạy calendar local: gRPC :9083 + gateway :8083
	bash scripts/run-service.sh calendar

run-calendar-grpc: ## Chỉ chạy gRPC server calendar (:9083)
	cd service/calendar && go run ./cmd/grpc

run-calendar-http: ## Chỉ chạy HTTP gateway calendar (:8083)
	cd service/calendar && go run ./cmd/http

run-mail-provider: ## Chạy mail-provider local: gRPC :9084 + gateway :8084
	bash scripts/run-service.sh mail-provider

run-mail-provider-grpc: ## Chỉ chạy gRPC server mail-provider (:9084)
	cd service/mail-provider && go run ./cmd/grpc

run-mail-provider-http: ## Chỉ chạy HTTP gateway mail-provider (:8084)
	cd service/mail-provider && go run ./cmd/http

web-install: ## Cài dependency frontend
	cd frontend && npm install

web-dev: error-codes ## Chạy frontend dev server
	cd frontend && npm run dev

web-build: error-codes ## Build frontend
	cd frontend && npm run build
