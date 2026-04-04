FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /workspace

RUN printf 'go 1.25\n\nuse (\n\t./pkg\n\t./services/submission\n)\n' > go.work

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/submission/go.mod ./services/submission/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd /workspace/pkg && go mod download && \
    cd /workspace/services/submission && go mod download

COPY pkg/ ./pkg/
COPY services/submission/ ./services/submission/

ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT_SHA

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    cd /workspace/services/submission && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -trimpath \
      -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.buildTime=${BUILD_TIME} \
        -X main.commitSHA=${COMMIT_SHA}" \
      -o /workspace/submission-service ./cmd/server

FROM alpine:3.21 AS runtime

RUN apk add --no-cache \
      tzdata \
      ca-certificates && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup -h /app -s /sbin/nologin

WORKDIR /app

RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

COPY --from=builder --chown=appuser:appgroup /workspace/submission-service ./app

USER appuser

EXPOSE 8083

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8083/health || exit 1

LABEL org.opencontainers.image.title="submission-service" \
      org.opencontainers.image.description="Submission management microservice for Go Judge System" \
      org.opencontainers.image.source="https://github.com/go-judge-system"

ENTRYPOINT ["./app"]