BIN := listmonk
STATIC := config.toml.sample schema.sql queries.sql public email-templates frontend/my/build:/frontend

HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH} (${COMMIT_DATE})

.PHONY: build-frontend
build-frontend:
	cd frontend/my && yarn install && yarn build

.PHONY: quickdev
quickdev:
	@ if [ ! -d "frontend/my/node_modules" ]; then \
		echo "Installing frontend deps"; \
		cd frontend/my && yarn install; \
	fi

	@ if [ ! -d "frontend/my/build" ]; then \
		echo "Creating build directory"; \
		mkdir -p frontend/my/build; \
		echo "Building frontend assets"; \
		cd frontend/my && yarn build; \
	fi

	@ echo -e "\nBuilding go binary\n"
	make build

	@ echo -e "Editing database params inside config\n"
	cp config.toml.sample config.toml

	@ echo -n "Database user: "
	@ read DBUSER; \
	sed -i -e "s/user = \"listmonk\"/user = \"$${DBUSER}\"/g" config.toml

	@ echo -n "Database password: "
	@ read DBPASSWORD; \
	sed -i -e "s/password = \"\"/password = \"$${DBPASSWORD}\"/g" config.toml

	@ echo -n "Database name: "
	@ read DBNAME; \
	sed -i -e "s/database = \"listmonk\"/database = \"$${DBNAME}\"/g" config.toml; \
	createdb $${DBNAME}

	@ echo -e "Running installer\n"
	./listmonk --install

.PHONY: build
build:
	go build  -o ${BIN} -ldflags="-s -w -X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'"
	stuffbin -a stuff -in ${BIN} -out ${BIN} ${STATIC}

.PHONY: run
run: build
	./${BIN}

.PHONY: deps
deps:
	go get -u github.com/knadh/stuffbin/...

.PHONY: test
test:
	go test

.PHONY: clean
clean:
	go clean
	- rm -f ${BIN}
