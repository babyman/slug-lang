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

# Help message
show_help() {
  echo "Usage: $(basename "$0") [options] [script]"
  echo ""
  echo "Options:"
  echo "  -h, --help       Show this help message and exit."
  echo "  -v, --volume     Mount a local directory (default: current directory)."
  echo "  --extra          Pass additional arguments to the tool."
  echo ""
  echo "Examples:"
  echo "  $(basename "$0") --help               # Display help information"
  echo "  $(basename "$0") myscript.slug        # Run a script with the tool"
  echo "  $(basename "$0") -v /my/data          # Mount /my/data into the container"
  exit 0
}

# Default volume path (current working directory)
VOLUME="$(pwd)"

# Parse arguments
EXTRA_ARGS=()
SCRIPT=""
VOLUME_SPECIFIED=false

while [[ "$#" -gt 0 ]]; do
  case $1 in
    -h|--help) show_help ;;
    -v|--volume) VOLUME="$2"; VOLUME_SPECIFIED=true; shift ;;
    --extra) shift; while [[ $# -gt 0 ]]; do EXTRA_ARGS+=("$1"); shift; done; break ;;
    *) SCRIPT="$1" ;;
  esac
  shift
done

# Check if user specifies a volume explicitly and validate it
if [[ "$VOLUME_SPECIFIED" == true && ! -d "$VOLUME" ]]; then
  echo "Error: Specified volume directory '$VOLUME' does not exist."
  exit 1
fi

# Check if input script is specified or needed
if [[ -n "$SCRIPT" && ! -f "$SCRIPT" ]]; then
  echo "Error: Script file '$SCRIPT' not found."
  exit 1
fi

# Execute container with the given script and volume
$CONTAINER_ENGINE run --rm -it \
  -v "$VOLUME:/data" \
  "$IMAGE_NAME" $SCRIPT "${EXTRA_ARGS[@]}"
