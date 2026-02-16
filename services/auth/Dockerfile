FROM golang:1.24-alpine AS builder

WORKDIR /workspace

COPY go.work .
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

WORKDIR /workspace/pkg
RUN go mod download

WORKDIR /workspace/services/auth
RUN go mod download

WORKDIR /workspace
COPY pkg/ ./pkg/
COPY services/auth/ ./services/auth/

WORKDIR /workspace/services/auth
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app cmd/main.go

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache tzdata ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /workspace/services/auth/app .

RUN chown appuser:appgroup /app/app

USER appuser

CMD ["./app"]