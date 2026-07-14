FROM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o reciperepo .

FROM debian:stable-slim
WORKDIR /app

RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/reciperepo ./reciperepo

CMD ["./reciperepo"]

