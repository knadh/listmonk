# Stage 1: Build Frontend Dependencies
FROM docker.io/library/node:22-alpine AS frontend-deps
WORKDIR /app/frontend

# Install yarn (though it's usually in node:alpine)
RUN apk add --no-cache yarn

# Copy package files for caching
COPY frontend/package.json frontend/yarn.lock* frontend/package-lock.json* ./
RUN yarn install --frozen-lockfile || yarn install

# Stage 2: Build Email Builder Dependencies
FROM docker.io/library/node:22-alpine AS email-builder-deps
WORKDIR /app/frontend/email-builder

RUN apk add --no-cache yarn
COPY frontend/email-builder/package.json frontend/email-builder/yarn.lock* frontend/email-builder/package-lock.json* ./
RUN yarn install --frozen-lockfile || yarn install

# Stage 3: Build Frontend Assets
FROM docker.io/library/node:22-alpine AS frontend-builder
WORKDIR /app

# Copy dependencies from cache stages
COPY --from=frontend-deps /app/frontend/node_modules ./frontend/node_modules
COPY --from=email-builder-deps /app/frontend/email-builder/node_modules ./frontend/email-builder/node_modules

# Copy source code
COPY frontend ./frontend
COPY static ./static
RUN touch frontend/.gitignore

# Build App Frontend
WORKDIR /app/frontend
RUN yarn build

# Build Email Builder
WORKDIR /app/frontend/email-builder
RUN yarn build

# Move Email Builder dist to final location expected by backend
WORKDIR /app
RUN mkdir -p frontend/public/static/email-builder && \
    cp -r frontend/email-builder/dist/* frontend/public/static/email-builder/

# Stage 4: Build Backend
FROM docker.io/library/golang:1.24-alpine AS backend-builder
WORKDIR /app

# Install stuffbin
RUN go install github.com/knadh/stuffbin/stuffbin@latest

# Copy go mod files for caching
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

# Stage 5: Final Runtime Image
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
