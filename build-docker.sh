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

# Build the Docker image using docker-compose
$DOCKER_COMPOSE build