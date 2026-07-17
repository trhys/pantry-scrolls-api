FROM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ps_server .

FROM debian:stable-slim
WORKDIR /app

RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y ca-certificates wget && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/ps_server ./ps_server

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

CMD ["./ps_server"]

