FROM node:19-alpine as front-builder

RUN apk add --no-cache alpine-sdk

WORKDIR /app
ADD . /app

RUN make build-frontend

FROM golang:alpine as builder

RUN apk add --no-cache alpine-sdk

WORKDIR /app
ADD . /app
COPY --from=front-builder /app/frontend /app/frontend

RUN make dist

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /listmonk

COPY --from=builder /app/listmonk .

COPY config.toml.sample config.toml
COPY config-demo.toml .
CMD ["./listmonk"]
EXPOSE 9000
