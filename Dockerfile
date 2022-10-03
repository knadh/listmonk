FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /listmonk
COPY listmonk .
COPY config.toml.sample config.toml
COPY config-demo.toml .
CMD ["./listmonk"]
EXPOSE 9000
