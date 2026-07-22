# syntax=docker/dockerfile:1

# ---- 前端构建 ----
FROM node:20-alpine AS frontend
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# ---- 后端构建 ----
FROM golang:1.25.7-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /src/web/dist ./static

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG BUILD_ENV=production

RUN CGO_ENABLED=1 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X 'llm-gateway/internal/core.BuildEnv=${BUILD_ENV}' -X 'llm-gateway/internal/core.BuildTime=${BUILD_TIME}' -X 'llm-gateway/internal/core.BuildVersion=${VERSION}'" \
    -o /out/lgw ./cmd/

# ---- 运行镜像 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S lgw && adduser -S -G lgw lgw

COPY --from=builder /out/lgw /usr/local/bin/lgw

RUN mkdir -p /data/db /data/logs && chown -R lgw:lgw /data

USER lgw
WORKDIR /data

VOLUME ["/data"]
EXPOSE 3001

ENTRYPOINT ["/usr/local/bin/lgw"]
CMD ["--http-addr", ":3001", "--data-dir", "/data"]