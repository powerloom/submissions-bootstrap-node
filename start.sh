#!/bin/bash

# Detect docker compose command (docker compose plugin vs docker-compose standalone)
if docker compose version >/dev/null 2>&1; then
	DOCKER_COMPOSE="docker compose"
elif docker-compose version >/dev/null 2>&1; then
	DOCKER_COMPOSE="docker-compose"
else
	echo "Error: Neither 'docker compose' nor 'docker-compose' found. Please install Docker Compose."
	exit 1
fi

# Trap SIGINT and SIGTERM to ensure proper cleanup
trap "echo 'Stopping bootstrap node...'; $DOCKER_COMPOSE down; exit 0" SIGINT SIGTERM

# If a port is supplied as the first argument, export it so that it overrides
# the value in .env. Otherwise, rely entirely on Docker Compose's .env handling.
if [ -n "$1" ]; then
  export BOOTSTRAP_PORT="$1"
fi

$DOCKER_COMPOSE up
