#!/usr/bin/env sh
set -eu

# Listmonk demo setup using `docker-compose`.
# See https://listmonk.app/docs/installation/ for detailed installation steps.

check_dependencies() {
	if ! command -v curl > /dev/null; then
		echo "curl is not installed."
		exit 1
	fi

	if ! command -v docker > /dev/null; then
		echo "docker is not installed."
		exit 1
	fi

	if ! command -v docker-compose > /dev/null; then
		echo "docker-compose is not installed."
		exit 1
	fi
}

setup_containers() {
	curl -o docker-compose.yml https://raw.githubusercontent.com/knadh/listmonk/master/docker-compose.yml
	docker-compose up -d demo-db demo-app
}

show_output(){
	echo -e "\nListmonk is now up and running. Visit http://localhost:9000 in your browser.\n"
}


check_dependencies
setup_containers
show_output
