FROM alpine:latest

# Install dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec

# Set working directory
WORKDIR /listmonk

# Copy binaries and configs
COPY listmonk run.sh config.toml config.toml.sample queries.sql schema.sql permissions.json ./
COPY docker-entrypoint.sh /usr/local/bin/

# Copy static assets
COPY static/ ./static/
COPY i18n/ ./i18n/
COPY frontend/dist/ ./frontend/dist/

# Make scripts executable
RUN chmod +x /usr/local/bin/docker-entrypoint.sh ./run.sh

EXPOSE 9000

ENTRYPOINT ["./run.sh"]
