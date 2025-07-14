#!/bin/bash

IMAGE_NAME="submissions-bootstrap-node"

# Remove existing image if it exists
docker rmi ${IMAGE_NAME} > /dev/null 2>&1 || true

# Build the Docker image with --no-cache to ensure a fresh build
docker build --no-cache -t ${IMAGE_NAME} .