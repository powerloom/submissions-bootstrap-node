#!/bin/bash

IMAGE_NAME="submissions-bootstrap-node"

# Build the Docker image
docker build -t ${IMAGE_NAME} .
