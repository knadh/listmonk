FROM oven/bun:1.4.3 AS bun

FROM golang:1.27.2-trixie

RUN apt-get update && apt-get install -y --no-install-recommends python3 \
    && rm -rf /var/lib/apt/lists/* \
    && git config --system --add safe.directory /app

COPY --from=bun /usr/local/bin/bun /usr/local/bin/bun
ENV CGO_ENABLED=0

WORKDIR /app
CMD ["sleep", "infinity"]
