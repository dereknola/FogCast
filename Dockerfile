# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM node:26-alpine AS web-dm
WORKDIR /src
COPY web/dm/package*.json ./web/dm/
RUN npm --prefix ./web/dm ci
COPY web/dm/ ./web/dm/
RUN npm --prefix ./web/dm run build

FROM --platform=$BUILDPLATFORM node:26-alpine AS web-player
WORKDIR /src
COPY web/player/package*.json ./web/player/
RUN npm --prefix ./web/player ci
COPY web/player/ ./web/player/
RUN npm --prefix ./web/player run build

FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.9.0 AS xx

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS builder
COPY --from=xx / /
ARG TARGETPLATFORM
WORKDIR /src

RUN apt-get update \
	&& apt-get install -y --no-install-recommends build-essential pkg-config clang lld \
	&& rm -rf /var/lib/apt/lists/*
RUN xx-apt-get update \
	&& xx-apt-get install -y --no-install-recommends xx-c-essentials libwebp-dev \
	&& rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN mkdir -p /data/maps
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=1 xx-go build -trimpath -ldflags="-s -w" -o /out/fogcast ./cmd/fogcast \
	&& xx-verify /out/fogcast

FROM gcr.io/distroless/base-debian13:nonroot AS runtime
WORKDIR /app
COPY --from=builder --chown=65532:65532 /data /data
COPY --from=builder --chown=65532:65532 /out/fogcast /app/fogcast
COPY --from=web-dm --chown=65532:65532 /src/static/dm/ /app/static/dm/
COPY --from=web-player --chown=65532:65532 /src/static/player/ /app/static/player/

ENV FOGCAST_ADDR=:8080
ENV FOGCAST_DATA_DIR=/data
ENV FOGCAST_STATIC_DIR=/app/static

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/app/fogcast"]
