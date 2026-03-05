# Stage 1: Build Frontend
FROM docker.io/library/node:22-alpine AS frontend-builder
WORKDIR /app

# Install yarn
RUN apk add --no-cache yarn

# Copy all frontend files
COPY frontend ./frontend
COPY static ./static

# Ensure an empty .gitignore exists for ESLint
RUN touch frontend/.gitignore

# Build App Frontend
WORKDIR /app/frontend
RUN yarn install && yarn build

# Build Email Builder
WORKDIR /app/frontend/email-builder
RUN yarn install && yarn build

# Move Email Builder dist to final location expected by backend
WORKDIR /app
RUN mkdir -p frontend/public/static/email-builder && \
    cp -r frontend/email-builder/dist/* frontend/public/static/email-builder/

# Stage 2: Build Backend
FROM docker.io/library/golang:1.24-alpine AS backend-builder
WORKDIR /app

# Install stuffbin
RUN go install github.com/knadh/stuffbin/stuffbin@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets from previous stage
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY --from=frontend-builder /app/frontend/public/static/email-builder ./frontend/public/static/email-builder

# Build binary
RUN CGO_ENABLED=0 go build -o listmonk -ldflags="-s -w" cmd/*.go

# Pack static assets into the binary using stuffbin
RUN /go/bin/stuffbin -a stuff -in listmonk -out listmonk \
    config.toml.sample \
    schema.sql queries:/queries permissions.json \
    static/public:/public \
    static/email-templates \
    frontend/dist:/admin \
    i18n:/i18n

# Stage 3: Final Runtime Image
FROM docker.io/library/alpine:latest
WORKDIR /listmonk

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec

# Copy the packed binary and necessary files
COPY --from=backend-builder /app/listmonk .
COPY config.toml.sample config.toml
COPY docker-entrypoint.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Version injection
ARG APP_VERSION=dev
ENV APP_VERSION=$APP_VERSION

EXPOSE 9000
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["./listmonk"]
