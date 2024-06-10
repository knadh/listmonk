FROM alpine:latest

# Set the maintainer information
LABEL org.opencontainers.image.title="listmonk"
LABEL org.opencontainers.image.description="listmonk is a standalone, self-hosted, newsletter and mailing list manager. It is fast, feature-rich, and packed into a single binary."
LABEL org.opencontainers.image.url="https://listmonk.app"
LABEL org.opencontainers.image.documentation='https://listmonk.app'
LABEL org.opencontainers.image.source='https://github.com/knadh/listmonk'
LABEL org.opencontainers.image.licenses='AGPL-3.0'

# Install dependencies
RUN apk --no-cache add ca-certificates tzdata shadow su-exec

# Set the working directory
WORKDIR /listmonk

# Copy only the necessary files
COPY listmonk .
COPY config.toml.sample config.toml
COPY config-demo.toml .

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
