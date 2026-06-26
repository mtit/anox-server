#!/usr/bin/env bash
# 构建 Anox Docker 镜像（仓库内不含 Dockerfile，由本脚本生成并构建）
#
# 用法:
#   ./scripts/build-docker-image.sh
#   ANOX_IMAGE=anox-server:1.0.0 ./scripts/build-docker-image.sh
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE="${ANOX_IMAGE:-anox-server:latest}"

cd "$ROOT_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "Error: docker not found"
  exit 1
fi

echo "==> Building image: ${IMAGE}"
echo "==> Context: ${ROOT_DIR}"

docker build -t "${IMAGE}" -f - . <<'EOF'
FROM node:20-alpine AS web-builder

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npx vite build


FROM golang:1.24-alpine AS go-builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY api/ ./api/
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

COPY --from=web-builder /src/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /anox-server ./cmd/anox-server


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=go-builder /anox-server /app/anox-server
COPY --from=go-builder /src/web/dist /app/web/dist

RUN mkdir -p /app/data/configs /app/logs /app/defaults/configs \
  && printf '%s\n' \
    '{' \
    '  "version": 1,' \
    '  "values": {' \
    '    "log_level": "info"' \
    '  }' \
    '}' \
    > /app/defaults/configs/_global.json \
  && printf '%s\n' \
    '#!/bin/sh' \
    'set -e' \
    'mkdir -p /app/data/configs /app/logs' \
    'if [ ! -f /app/data/configs/_global.json ]; then' \
    '  cp /app/defaults/configs/_global.json /app/data/configs/_global.json' \
    'fi' \
    'exec /app/anox-server' \
    > /app/entrypoint.sh \
  && chmod +x /app/entrypoint.sh

ENV HOST=0.0.0.0 \
    PORT=8848 \
    TZ=Asia/Shanghai

EXPOSE 8848

VOLUME ["/app/data", "/app/logs"]

ENTRYPOINT ["/app/entrypoint.sh"]
EOF

echo ""
echo "==> Build complete: ${IMAGE}"
echo ""
echo "启动示例（按需修改参数）:"
echo ""
cat <<EXAMPLE
# 确保网络存在
docker network inspect dev-net >/dev/null 2>&1 || docker network create dev-net

# 首次或更新后启动
docker rm -f anox-server 2>/dev/null || true

docker run -d \\
  --name anox-server \\
  --restart unless-stopped \\
  --network dev-net \\
  -p 8848:8848 \\
  -e HOST=0.0.0.0 \\
  -e PORT=8848 \\
  -e PASS=your-password \\
  -v anox-data:/app/data \\
  -v anox-logs:/app/logs \\
  ${IMAGE}
EXAMPLE
echo ""
echo "同网服务访问: http://anox-server:8848  ws://anox-server:8848/ws"
echo "浏览器访问:   http://<服务器IP>:8848"
