# Build stage
FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY processor/ ./processor/
WORKDIR /app/processor
RUN go mod tidy

ENV CGO_ENABLED=0
RUN go build -o app

# ---
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/processor/app .

ENTRYPOINT ["./app"]
