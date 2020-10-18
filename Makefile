LAST_COMMIT := $(shell git rev-parse --short HEAD)
VERSION := $(shell git describe --tags --abbrev=0)
BUILDSTR := ${VERSION} (\#${LAST_COMMIT} $(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))

BIN := listmonk
STATIC := config.toml.sample \
	schema.sql queries.sql \
	static/public:/public \
	static/email-templates \
	frontend/dist/favicon.png:/frontend/favicon.png \
	frontend/dist/frontend:/frontend

# Install dependencies for building.
.PHONY: deps
deps:
	go get -u github.com/knadh/stuffbin/...
	cd frontend && yarn install

# Build the backend to ./listmonk.
.PHONY: build
build:
	go build -o ${BIN} -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}'" cmd/*.go

# Run the backend.
.PHONY: run
run: build
	./${BIN}

# Build the JS frontend into frontend/dist.
.PHONY: build-frontend
build-frontend:
	export VUE_APP_VERSION="${VERSION}" && cd frontend && yarn build

# Run the JS frontend server in dev mode.
.PHONY: run-frontend
run-frontend:
	export VUE_APP_VERSION="${VERSION}" && cd frontend && yarn serve

# Run Go tests.
.PHONY: test
test:
	go test ./...

# Bundle all static assets including the JS frontend into the ./listmonk binary
# using stuffbin (installed with make deps).
.PHONY: dist
dist: build build-frontend
	stuffbin -a stuff -in ${BIN} -out ${BIN} ${STATIC}

# pack-releases runns stuffbin packing on the given binary. This is used
# in the .goreleaser post-build hook.
.PHONY: pack-bin
pack-bin:
	stuffbin -a stuff -in $(bin) -out $(bin) ${STATIC}

# Use goreleaser to do a dry run producing local builds.
.PHONY: release-dry
release-dry:
	goreleaser --parallelism 1 --rm-dist --snapshot --skip-validate --skip-publish

# Use goreleaser to build production releases and publish them.
.PHONY: release
release:
	goreleaser --parallelism 1 --rm-dist --skip-validate
