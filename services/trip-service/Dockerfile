# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /trip-service ./services/trip-service

FROM alpine:3.20
RUN adduser -D -H appuser
USER appuser
WORKDIR /home/appuser
COPY --from=builder /trip-service ./trip-service
EXPOSE 8081
ENTRYPOINT ["./trip-service"]
