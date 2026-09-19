FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api ./cmd/api
RUN go build -o migrate ./cmd/migrate

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /app/api .
COPY --from=build /app/migrate .
COPY --from=build /app/migrations ./migrations

EXPOSE 8080

CMD ["sh", "-c", "./migrate up && exec ./api"]