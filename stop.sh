#!/usr/bin/env bash

set -e

export COMPOSE_PROJECT_NAME=negentropy

vault_counts=$(docker ps --format '{{.Names}}' | grep vault | wc -l)
if (( $vault_counts < 2 )); then
  echo "DEBUG: found single vault instance"
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.dev.yml down
else
  echo "DEBUG: found two separate vaults"
  # no matter which start option was chosen (e2e, local or debug) - container's names don't change
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml down
fi
