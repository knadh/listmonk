FROM node:16 as base

WORKDIR /listmonk
COPY . .
RUN wget https://golang.org/dl/go1.17.1.linux-amd64.tar.gz
RUN cat static/public/templates/index.html
ARG VUE_APP_ROOT_URL
ARG LISTMONK_FRONTEND_ROOT
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.1.linux-amd64.tar.gz && export PATH=$PATH:/usr/local/go/bin && make dist


FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /listmonk
COPY --from=base listmonk .
COPY static static
COPY config.toml.sample config.toml
COPY config-demo.toml .
CMD ["./listmonk"]
EXPOSE 9000
