# Build stage
FROM golang:1.24-alpine AS build
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/app ./cmd/api

# Runtime stage
FROM alpine:3.20
WORKDIR /app

RUN adduser -D -h /app appuser
COPY --from=build /app/bin/app /app/app

RUN mkdir -p /app/uploads && chown -R appuser:appuser /app

ENV ADDR=0.0.0.0:8080
EXPOSE 8080

USER appuser
ENTRYPOINT ["/app/app"]
