# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app

# Match your existing build env
ENV CGO_ENABLED=0 GOOS=linux
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOPRIVATE=github.com/klauspost/*
ENV GONOSUMDB=github.com/klauspost/*

# Go deps
COPY go.mod go.sum ./
RUN go mod download

# Copy only what we need
COPY pkg ./pkg
COPY services/matching-service ./services/matching-service

# Build
RUN go build -ldflags="-s -w" -o /matching-service ./services/matching-service

# --- Runtime image ---
FROM alpine:3.20
RUN adduser -D -H appuser
USER appuser
WORKDIR /home/appuser

# (Optional) add CA bundle for HTTPS if you’ll ever call TLS endpoints
# RUN apk add --no-cache ca-certificates

COPY --from=builder /matching-service ./matching-service
EXPOSE 8083
ENTRYPOINT ["./matching-service"]
