FROM golang:1.24.1 AS go

FROM node:16 AS node

COPY --from=go /usr/local/go /usr/local/go
ENV GOPATH /go
ENV CGO_ENABLED=0
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /app
CMD [ "sleep infinity" ]
