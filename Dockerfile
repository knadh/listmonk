# Stage 1: Build Frontend Dependencies
FROM docker.io/library/node:22-alpine AS frontend-deps
WORKDIR /app/frontend

RUN apk add --no-cache yarn
RUN mkdir -p /app/static/public/static
COPY frontend/package.json frontend/yarn.lock* frontend/package-lock.json* ./
RUN yarn install --frozen-lockfile || yarn install

# Stage 2: Build Email Builder Dependencies
FROM docker.io/library/node:22-alpine AS email-builder-deps
WORKDIR /app/frontend/email-builder

RUN apk add --no-cache yarn
COPY frontend/email-builder/package.json frontend/email-builder/yarn.lock* frontend/email-builder/package-lock.json* ./
RUN yarn install --frozen-lockfile || yarn install

# Stage 3: Build Assets
FROM docker.io/library/node:22-alpine AS builder
WORKDIR /app

# Copy dependencies
COPY --from=frontend-deps /app/frontend/node_modules ./frontend/node_modules
COPY --from=email-builder-deps /app/frontend/email-builder/node_modules ./frontend/email-builder/node_modules

# Copy source code
COPY frontend ./frontend
COPY static ./static
RUN touch frontend/.gitignore

# 1. Build Email Builder FIRST (as required by the main frontend build)
WORKDIR /app/frontend/email-builder
RUN yarn build

# 2. Move Email Builder dist to the public folder of the main frontend
WORKDIR /app
RUN mkdir -p frontend/public/static/email-builder && \
    cp -r frontend/email-builder/dist/* frontend/public/static/email-builder/

# 3. Build App Frontend
WORKDIR /app/frontend
RUN yarn build

# Stage 4: Build Backend
FROM docker.io/library/golang:1.24-alpine AS backend-builder
WORKDIR /app

RUN go install github.com/knadh/stuffbin/stuffbin@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Copy built frontend assets (dist now contains email-builder too)
COPY --from=builder /app/frontend/dist ./frontend/dist

# Build binary
RUN CGO_ENABLED=0 go build -o listmonk -ldflags="-s -w" cmd/*.go

# Pack static assets
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

RUN apk --no-cache add ca-certificates tzdata shadow su-exec
COPY --from=backend-builder /app/listmonk .
COPY config.toml.sample config.toml
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ARG APP_VERSION=dev
ENV APP_VERSION=$APP_VERSION

EXPOSE 9000
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["sh", "-c", "./listmonk --install --idempotent --yes --config '' && ./listmonk --upgrade --yes --config '' && ./listmonk --config ''"]
