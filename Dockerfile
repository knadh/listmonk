FROM alpine:latest AS deploy
RUN apk --no-cache add ca-certificates
COPY listmonk /
COPY config.toml.sample  /etc/listmonk/config.toml
VOLUME ["/etc/listmonk"]
CMD ["./listmonk", "--config", "/etc/listmonk/config.toml"]  
