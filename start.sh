#!/usr/bin/env bash

set -e


while [[ $# -gt 0 ]]; do
  case $1 in
    -d|--debug)
    export DEBUG="true"
    ;;
    *)
    echo "Unknown parameter $1"; exit 1
    ;;
  esac
  shift
done

export COMPOSE_PROJECT_NAME=negentropy

if [[ $DEBUG == "true" ]]; then
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.debug.yml up -d
  echo "DEBUG: run you debug tools and connect to dlv servers at both vaults"
  while true; do
    read -p "Are you ready? (Y/n) " ANSWER;
      if [[ -z "$ANSWER" ]]; then ANSWER=Y; fi
      case $ANSWER in
          [Yy]* ) break;;
          [Nn]* ) exit 1;;
      esac
  done
else
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml up -d
  echo "DEBUG: sleep for 5s while containers starts"
  sleep 5
fi

python --version
python3 --version
pip install virtualenv
virtualenv scripts/e2e
source scripts/e2e/bin/activate
pip install -r scripts/requirements.txt
python scripts/start.py

deactivate

