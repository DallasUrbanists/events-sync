FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM builder AS events-sync-build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/events-sync ./cmd/events-sync

FROM builder AS migrate-build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/migrate ./cmd/migrate

FROM builder AS web-build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/web ./cmd/web

FROM alpine:latest AS runner
RUN apk --no-cache add ca-certificates
WORKDIR /root/

FROM runner AS events-sync
COPY --from=events-sync-build /app/bin bin
COPY --from=events-sync-build /app/config.json .

FROM runner AS migrate
COPY --from=migrate-build /app/bin bin
COPY --from=migrate-build /app/config.json .

FROM runner AS web
COPY --from=web-build /app/bin bin
COPY --from=web-build /app/config.json .
COPY --from=web-build /app/web web
