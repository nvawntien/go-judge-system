FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /workspace

RUN printf 'go 1.24.0\n\nuse (\n\t./pkg\n\t./services/problem\n)\n' > go.work

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/problem/go.mod services/problem/go.sum ./services/problem/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd /workspace/pkg && go mod download && \
    cd /workspace/services/problem && go mod download


COPY pkg/ ./pkg/
COPY services/problem/ ./services/problem/


ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT_SHA

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    cd /workspace/services/problem && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -trimpath \
      -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.buildTime=${BUILD_TIME} \
        -X main.commitSHA=${COMMIT_SHA}" \
      -o /workspace/problem-service ./cmd/server

FROM alpine:3.21 AS runtime

RUN apk add --no-cache \
      tzdata \
      ca-certificates && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup -h /app -s /sbin/nologin

WORKDIR /app

RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

COPY --from=builder --chown=appuser:appgroup /workspace/problem-service ./app

USER appuser

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8082/health || exit 1

LABEL org.opencontainers.image.title="problem-service" \
      org.opencontainers.image.description="Problem management microservice for Go Judge System" \
      org.opencontainers.image.source="https://github.com/go-judge-system"

ENTRYPOINT ["./app"]
