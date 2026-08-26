# ---- 构建阶段 ----
FROM golang:1.23-alpine AS build
WORKDIR /src

# 先复制模块定义，充分利用 Docker 层缓存。
COPY go.mod ./
# 项目无第三方依赖，因此无需 go mod download；直接复制全部源码与内嵌前端。
COPY . .

# 跟随构建平台（BuildKit 自动注入 TARGETOS/TARGETARCH），避免在 arm64 宿主机上产出
# 无法运行的 amd64 二进制；如需固定架构，可在 docker build 时 --build-arg TARGETARCH=amd64。
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath -ldflags="-s -w" \
    -o /out/boiler-energy-efficiency-service .

# ---- 运行阶段 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app appuser
WORKDIR /app
COPY --from=build /out/boiler-energy-efficiency-service /app/boiler-energy-efficiency-service
RUN chown -R appuser:app /app
USER appuser

ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT:-8080}/healthz" || exit 1

ENTRYPOINT ["/app/boiler-energy-efficiency-service"]
