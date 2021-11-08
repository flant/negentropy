#!/usr/bin/env bash

set -e

export COMPOSE_PROJECT_NAME=negentropy

# no matter which start option was chosen (normal or debug) - container's names don't change
docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml down
