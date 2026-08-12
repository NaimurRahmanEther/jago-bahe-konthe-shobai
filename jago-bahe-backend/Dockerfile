# Multi-stage build producing small static binaries for both the api and worker.
FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
# Run as an unprivileged user, not root.
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
USER app
EXPOSE 8080
# docker-compose overrides the command for the worker service.
ENTRYPOINT ["/app/api"]
