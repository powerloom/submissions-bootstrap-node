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

# Parse arguments
BUILD_IMAGE=false
PORT_ARG=""

while [ $# -gt 0 ]; do
	case "$1" in
		--build)
			BUILD_IMAGE=true
			shift
			;;
		*)
			# Treat any non-flag argument as a port number
			PORT_ARG="$1"
			shift
			;;
	esac
done

# Build image if requested
if [ "$BUILD_IMAGE" = true ]; then
	echo "Building Docker image..."
	$DOCKER_COMPOSE build
	if [ $? -ne 0 ]; then
		echo "Error: Docker build failed"
		exit 1
	fi
	echo "Build complete"
fi

# Trap SIGINT and SIGTERM to ensure proper cleanup
trap "echo 'Stopping bootstrap node...'; $DOCKER_COMPOSE down; exit 0" SIGINT SIGTERM

# If a port is supplied, export it so that it overrides the value in .env
if [ -n "$PORT_ARG" ]; then
	export BOOTSTRAP_PORT="$PORT_ARG"
fi

$DOCKER_COMPOSE up
