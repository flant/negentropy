#!/usr/bin/env bash

PATH="$PATH:$HOME/.local/bin"

set -e

while [[ $# -gt 0 ]]; do
  case $1 in
    --debug)
    export DEBUG="true"
    ;;
    --dev)
    export DEV="true"
    ;;
    *)
    echo "Unknown parameter $1"; exit 1
    ;;
  esac
  shift
done

export COMPOSE_PROJECT_NAME=negentropy

if [[ $DEV ==  "true" ]]; then
    echo "=========================================="
    echo "MODE: one vault in dev mode, plugins aside"
    echo "=========================================="
    export MODE="DEV"
    docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.dev.yml up -d
else
  if [[ $DEBUG == "true" ]]; then
    echo "============================================"
    echo "MODE: vaults with plugins onboard with debug"
    echo "============================================"
    export MODE="DEBUG"
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
    echo "================================="
    echo "MODE: vaults with plugins onboard"
    echo "================================="
    export MODE="NORMAL"
    docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml up -d
    echo "DEBUG: sleep for 5s while containers starts"
    sleep 5
  fi
fi

python3 --version
pip3 install virtualenv
virtualenv scripts/e2e
source scripts/e2e/bin/activate
pip3 install -r scripts/requirements.txt
python3 scripts/start.py $MODE

deactivate
