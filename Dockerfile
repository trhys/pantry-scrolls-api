FROM golang:latest AS builder
WORKDIR /app
COPY . .
RUN go build -o reciperepo .
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM debian:stable-slim
WORKDIR /app
COPY --from=builder app/reciperepo reciperepo
COPY --from=builder /go/bin/goose goose

COPY sql/schema sql/schema
COPY entrypoint.sh entrypoint.sh
RUN chmod +x entrypoint.sh

RUN apt update
RUN apt upgrade -y

CMD ["./entrypoint.sh"]
