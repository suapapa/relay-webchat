# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go .

ENV CGO_ENABLED=0
RUN go build -o relay ./relay

# ---
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/relay/relay .

EXPOSE 8080

CMD ["./relay", "-addr", ":8080"]
