# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY relay/ ./relay/
WORKDIR /app/relay
RUN go mod tidy

ENV CGO_ENABLED=0
RUN go build -o app

# ---
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/relay/app .

EXPOSE 8080

ENTRYPOINT ["./app"]
