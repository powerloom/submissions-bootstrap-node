#!/bin/sh
set -e

# Use BOOTSTRAP_PORT if set, otherwise default to 4001
PORT="${BOOTSTRAP_PORT:-4001}"

echo "Starting submissions bootstrap node on port ${PORT}"
exec ./submissions-bootstrap-node --port="${PORT}" 