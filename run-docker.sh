#!/bin/bash

# Default port
PORT=${1:-4001}

CONTAINER_NAME="submissions-bootstrap-node-instance"
IMAGE_NAME="submissions-bootstrap-node"

# Stop and remove any existing container with the same name
docker stop ${CONTAINER_NAME} > /dev/null 2>&1 || true
docker rm ${CONTAINER_NAME} > /dev/null 2>&1 || true

echo "🚀 Running ${IMAGE_NAME} on port ${PORT}..."

docker run -d -p ${PORT}:4001 --name ${CONTAINER_NAME} ${IMAGE_NAME} --port=4001

# Give the container a moment to start up
sleep 3

# Get the container's logs to find the multiaddress
LOGS=$(docker logs ${CONTAINER_NAME})

# Extract the Peer ID
PEER_ID=$(echo "$LOGS" | grep "Libp2p host created with ID:" | awk '{print $NF}')

# Construct the full multiaddress
MULTIADDR="/ip4/127.0.0.1/tcp/${PORT}/p2p/${PEER_ID}"

echo "✅ Bootstrap node is running."
echo "🌍 Multiaddress for other nodes: ${MULTIADDR} (replace 127.0.0.1 with current remote host's public IP address)"
