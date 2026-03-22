FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /workspace

RUN printf 'go 1.24.0\n\nuse (\n\t./pkg\n\t./workers/judge\n)\n' > go.work

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY workers/judge/go.mod ./workers/judge/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd /workspace/pkg && go mod download && \
    cd /workspace/workers/judge && go mod download

COPY pkg/ ./pkg/
COPY workers/judge/ ./workers/judge/

ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT_SHA

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    cd /workspace/workers/judge && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -trimpath \
      -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.buildTime=${BUILD_TIME} \
        -X main.commitSHA=${COMMIT_SHA}" \
      -o /workspace/judge-worker ./cmd/server

FROM alpine:3.21 AS runtime

RUN apk add --no-cache \
      tzdata \
      ca-certificates && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup -h /app -s /sbin/nologin

WORKDIR /app

RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

COPY --from=builder --chown=appuser:appgroup /workspace/judge-worker ./app

USER appuser

LABEL org.opencontainers.image.title="judge-worker" \
      org.opencontainers.image.description="Code Execution Worker for Go Judge System" \
      org.opencontainers.image.source="https://github.com/go-judge-system"

ENTRYPOINT ["./app"]
