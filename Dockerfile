# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o events-sync ./cmd/events-sync
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web ./cmd/web

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binaries from builder stage
COPY --from=builder /app/events-sync .
COPY --from=builder /app/web .

# Copy config file and web files
COPY --from=builder /app/config.json .
COPY --from=builder /app/web ./web-static

# Expose port (if needed for future web interface)
EXPOSE 8080

# Run the application
CMD ["./events-sync"]