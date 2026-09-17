#!/usr/bin/env bash
# Cài buf + các protoc plugin cần cho việc generate code.
# Yêu cầu: Go >= 1.24, curl. Không cần cài protoc (buf tự compile .proto).
set -euo pipefail

GOBIN_DIR="${GOBIN:-$(go env GOPATH)/bin}"
mkdir -p "$GOBIN_DIR"

log() { printf '\033[36m==>\033[0m %s\n' "$*"; }

# ---------------------------------------------------------------- buf
if command -v buf >/dev/null 2>&1; then
  log "buf đã có: $(buf --version)"
else
  case "$(uname -m)" in
    x86_64 | amd64) BUF_ARCH=x86_64 ;;
    aarch64 | arm64) BUF_ARCH=aarch64 ;;
    *) echo "Kiến trúc không hỗ trợ: $(uname -m)" >&2; exit 1 ;;
  esac
  BUF_VERSION="${BUF_VERSION:-latest}"
  log "Tải buf ($BUF_VERSION) cho linux-$BUF_ARCH"
  curl -fsSL -o "$GOBIN_DIR/buf" \
    "https://github.com/bufbuild/buf/releases/${BUF_VERSION}/download/buf-Linux-${BUF_ARCH}"
  chmod +x "$GOBIN_DIR/buf"
  log "buf đã cài vào $GOBIN_DIR/buf"
fi

# ------------------------------------------------------- protoc plugins
log "Cài protoc plugins vào $GOBIN_DIR"
GOBIN="$GOBIN_DIR" go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
GOBIN="$GOBIN_DIR" go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
GOBIN="$GOBIN_DIR" go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
GOBIN="$GOBIN_DIR" go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

log "Xong. Nhớ export PATH=\"$GOBIN_DIR:\$PATH\" rồi chạy: make gen"
ls -1 "$GOBIN_DIR" | grep -E '^(buf|protoc-gen-)' || true
