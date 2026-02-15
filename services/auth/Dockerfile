FROM golang:1.24-alpine AS builder
WORKDIR /workspace

COPY go.work .
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
RUN go work sync

COPY pkg/ ./pkg/
COPY services/auth/ ./services/auth/

WORKDIR /workspace/services/auth
RUN CGO_ENABLED=0 GOOS=linux go build -o app cmd/main.go

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache tzdata

COPY --from=builder /workspace/services/auth/app .

CMD ["./app"]