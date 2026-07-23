# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

ENV GOPROXY=https://goproxy.io,direct

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/fuck-watermark ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 10001 app

WORKDIR /app

COPY --from=builder /out/fuck-watermark /app/fuck-watermark

# 日志目录需在切用户前创建并授权，否则 lumberjack 无法 mkdir
RUN mkdir -p /app/logs \
    && chown -R app:app /app

USER app

EXPOSE 8080

ENTRYPOINT ["/app/fuck-watermark"]
CMD ["-c", "/app/config.toml"]
