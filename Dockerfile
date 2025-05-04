# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
COPY relay/ ./relay/
RUN go mod tidy
WORKDIR /app/relay

ENV CGO_ENABLED=0
RUN go build -o app

# ---
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/relay/app .

EXPOSE 8080

CMD ["./app", "-addr", ":8080"]
