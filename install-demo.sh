#!/usr/bin/env sh
set -eu

# Listmonk demo setup using `docker compose`.
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

	# Check for "docker compose" functionality.
	if ! docker compose version > /dev/null 2>&1; then
		echo "'docker compose' functionality is not available. Please update to a newer version of Docker. See https://docs.docker.com/engine/install/ for more details."
		exit 1
	fi
}

setup_containers() {
	curl -o docker-compose.yml https://raw.githubusercontent.com/knadh/listmonk/master/docker-compose.yml
	# Use "docker compose" instead of "docker-compose"
	docker compose up -d demo-db demo-app
}

show_output(){
	echo -e "\nListmonk is now up and running. Visit http://localhost:9000 in your browser.\n"
}


check_dependencies
setup_containers
show_output
