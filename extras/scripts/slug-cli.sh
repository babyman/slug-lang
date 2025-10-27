#!/bin/bash

# Wrapper script for the slug containerized CLI
if command -v podman >/dev/null 2>&1; then
  CONTAINER_ENGINE="podman"
elif command -v docker >/dev/null 2>&1; then
  CONTAINER_ENGINE="docker"
else
  echo "Error: Neither podman nor docker is installed."
  exit 1
fi

# Image name for the container
IMAGE_NAME="slug"

# Default volume path (current working directory)
VOLUME="$(pwd)"

# Execute container with the given script and volume
$CONTAINER_ENGINE run --rm -it \
  -v "$VOLUME:/data" \
  "$IMAGE_NAME" "$@"
