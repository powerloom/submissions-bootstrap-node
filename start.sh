#!/bin/bash

# If a port is supplied as the first argument, export it so that it overrides
# the value in .env. Otherwise, rely entirely on Docker Compose’s .env handling.
if [ -n "$1" ]; then
  export BOOTSTRAP_PORT="$1"
fi

echo "Starting bootstrap node (port: ${BOOTSTRAP_PORT:-<from .env>})"

docker-compose up
