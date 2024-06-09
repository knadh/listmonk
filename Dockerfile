# Use the latest alpine image as the base image
FROM alpine:latest

# Install dependencies
RUN apk --no-cache add ca-certificates tzdata

# Default values for UID and GID
ARG UID=1001
ARG GID=1001

# Create and set a non-root user
RUN addgroup -g ${GID} app && adduser -u ${UID} -G app -h /home/app -s /sbin/nologin -D app
USER app

# Set the working directory
WORKDIR /listmonk

# Copy only the necessary files
COPY --chown=app:app listmonk .
COPY --chown=app:app config.toml.sample config.toml
COPY --chown=app:app config-demo.toml .

# Expose the application port
EXPOSE 9000

# Define the command to run the application
CMD ["./listmonk"]
