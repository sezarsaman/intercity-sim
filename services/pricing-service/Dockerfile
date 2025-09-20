# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app

ENV CGO_ENABLED=0 GOOS=linux
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOPRIVATE=github.com/klauspost/*
ENV GONOSUMDB=github.com/klauspost/*

COPY go.mod go.sum ./
RUN go mod download

COPY pkg ./pkg
COPY services/pricing-service ./services/pricing-service

RUN go build -ldflags="-s -w" -o /pricing-service ./services/pricing-service

FROM alpine:3.20
RUN adduser -D -H appuser
USER appuser
WORKDIR /home/appuser
COPY --from=builder /pricing-service ./pricing-service
EXPOSE 8082
ENTRYPOINT ["./pricing-service"]
