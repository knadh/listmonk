# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies including Node.js for frontend
RUN apk add --no-cache git make build-base nodejs npm yarn

# Set the working directory
WORKDIR /src

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the frontend
WORKDIR /src/frontend
RUN yarn install --ignore-engines
# Skip prebuild (eslint) and run vite build directly
RUN yarn vite build

# Build the Go application
WORKDIR /src
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o listmonk ./cmd

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec

# Set the working directory
WORKDIR /listmonk

# Copy the binary from builder stage
COPY --from=builder /src/listmonk .

# Copy all necessary files from the builder stage
COPY --from=builder /src/config.toml.sample ./
COPY --from=builder /src/queries.sql ./
COPY --from=builder /src/schema.sql ./
COPY --from=builder /src/permissions.json ./
COPY --from=builder /src/static/ ./static/
COPY --from=builder /src/i18n/ ./i18n/
COPY --from=builder /src/frontend/dist/ ./frontend/dist/

# Create config.toml from sample
RUN cp config.toml.sample config.toml

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
