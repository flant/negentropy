#!/usr/bin/env bash

set -e

export COMPOSE_PROJECT_NAME=negentropy

docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.dev.yml down
