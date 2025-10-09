# -----------------------------------------------------
# Build stage
# -----------------------------------------------------
FROM golang:1.24.1-alpine AS builder

# Install build tools
RUN apk add --no-cache git build-base

# Set working directory
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary (note: main.go is in /cmd, not /cmd/listmonk)
RUN go build -o listmonk ./cmd

# -----------------------------------------------------
# Runtime stage
# -----------------------------------------------------
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec

# Set working directory
WORKDIR /listmonk

# Copy binary and config
COPY --from=builder /app/listmonk .
COPY config.toml.sample config.toml

# Expose application port
EXPOSE 9000

# Run
CMD ["./listmonk"]
