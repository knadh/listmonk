FROM alpine:latest AS deploy
RUN apk --no-cache add ca-certificates
WORKDIR /listmonk
COPY listmonk .
COPY config.toml.sample config.toml
CMD ["./listmonk"]
