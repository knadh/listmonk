BIN := listmonk
STATIC := config.toml.sample schema.sql queries.sql public email-templates frontend/my/build:/frontend

HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH} (${COMMIT_DATE})

build:
	go build  -o ${BIN} -ldflags="-s -w -X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'"
	stuffbin -a stuff -in ${BIN} -out ${BIN} ${STATIC}

deps:
	go get -u github.com/knadh/stuffbin/...

test:
	go test

clean:
	go clean
	- rm -f ${BIN}
