# Build stage
FROM golang:alpine AS builder

# Install build dependencies
RUN apk --no-cache add git make nodejs npm yarn

# Set the working directory
WORKDIR /app

# Copy source code
COPY . .

# Build the application
RUN go install github.com/knadh/stuffbin/... && \
    make dist

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec

# Set the working directory
WORKDIR /listmonk

# Copy only the necessary files from builder
COPY --from=builder /app/listmonk .
COPY --from=builder /app/config.toml.sample config.toml

# Copy the entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/

# Make the entrypoint script executable
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Expose the application port
EXPOSE 9000

# Set the entrypoint
ENTRYPOINT ["docker-entrypoint.sh"]

# Define the command to run the application
CMD ["./listmonk"]
