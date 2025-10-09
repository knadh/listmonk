# -----------------------------------------------------
# Build stage
# -----------------------------------------------------
FROM golang:1.24.1-alpine AS builder

# Install build tools, Git, Node.js, npm
RUN apk add --no-cache git build-base nodejs npm

# Set working directory
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# -----------------------------------------------------
# Build static frontend
# -----------------------------------------------------
WORKDIR /app/frontend

# Ensure a .gitignore exists (it's excluded from build context by .dockerignore).
# Then install Yarn v1, install dependencies and build the frontend.
RUN printf "" > .gitignore && \
    npm install -g yarn@1 && \
    (yarn install --frozen-lockfile || yarn install) && \
    yarn build

# -----------------------------------------------------
# Build backend binary with embedded static assets
# -----------------------------------------------------
WORKDIR /app
RUN go build -o listmonk ./cmd

# -----------------------------------------------------
# Runtime stage
# -----------------------------------------------------
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec

# Set working directory
WORKDIR /listmonk

# Copy backend binary and config file
COPY --from=builder /app/listmonk ./
COPY config.toml.sample ./config.toml

# Expose application port
EXPOSE 9000

# Run the app
CMD ["./listmonk"]
