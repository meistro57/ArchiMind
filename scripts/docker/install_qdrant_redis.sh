#!/usr/bin/env bash
# scripts/docker/install_qdrant_redis.sh
set -euo pipefail

# This script starts local Qdrant and Redis containers for ArchiMind development.
# Logical professional default: expose default ports and persist data in named volumes.

QDRANT_CONTAINER="archimind-qdrant"
REDIS_CONTAINER="archimind-redis"
QDRANT_IMAGE="qdrant/qdrant:latest"
REDIS_IMAGE="redis:7-alpine"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is not installed. Kindly install Docker first; this script is good, not magical."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not running. Give it a nudge and try again."
  exit 1
fi

echo "Pulling fresh images (because stale images are so last quarter)..."
docker pull "$QDRANT_IMAGE"
docker pull "$REDIS_IMAGE"

if docker ps -a --format '{{.Names}}' | grep -qx "$QDRANT_CONTAINER"; then
  echo "Reusing existing $QDRANT_CONTAINER container."
  docker start "$QDRANT_CONTAINER" >/dev/null
else
  echo "Creating $QDRANT_CONTAINER on port 6333."
  docker run -d \
    --name "$QDRANT_CONTAINER" \
    -p 6333:6333 \
    -v archimind_qdrant_data:/qdrant/storage \
    --restart unless-stopped \
    "$QDRANT_IMAGE" >/dev/null
fi

if docker ps -a --format '{{.Names}}' | grep -qx "$REDIS_CONTAINER"; then
  echo "Reusing existing $REDIS_CONTAINER container."
  docker start "$REDIS_CONTAINER" >/dev/null
else
  echo "Creating $REDIS_CONTAINER on port 6379."
  docker run -d \
    --name "$REDIS_CONTAINER" \
    -p 6379:6379 \
    -v archimind_redis_data:/data \
    --restart unless-stopped \
    "$REDIS_IMAGE" >/dev/null
fi

echo "Done. Qdrant: http://localhost:6333 | Redis: localhost:6379"
