FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/events-sync ./cmd/events-sync
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/web ./cmd/web

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/bin bin

COPY --from=builder /app/config.json .
COPY --from=builder /app/web web
