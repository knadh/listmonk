BIN := listmonk
STATIC := config.toml.sample schema.sql queries.sql public email-templates frontend/build:/frontend

HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH} (${COMMIT_DATE})

# Dependencies.
.PHONY: deps
deps:
	go get -u github.com/knadh/stuffbin/...
	cd frontend && yarn install

# Build steps.
.PHONY: build
build:
	go build  -o ${BIN} -ldflags="-s -w -X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'"

.PHONY: build-frontend
build-frontend:
	cd frontend && yarn build

.PHONY: dist
build-dist:
	stuffbin -a stuff -in ${BIN} -out ${BIN} ${STATIC}

.PHONY: run
run: build
	./${BIN}

.PHONY: run-frontend
run-frontend:
	cd frontend && yarn start

.PHONY: test
test:
	go test

.PHONY: clean
clean:
	go clean
	- rm -f ${BIN}
