#!/bin/bash

set -e

export PATH="$PATH:$HOME/.local/bin"

export COMPOSE_PROJECT_NAME="negentropy"

function usage {
  echo -e "Usage: $(basename "$0") [option]\n"
  echo "Options:"
  echo "  e2e		for e2e tests"
  echo "  single	single vault"
  echo "  debug		debug with dlv"
  echo "  local		local development"
}

case "$1" in
  single)
    export MODE="single"
    ;;
  e2e)
    export MODE="e2e"
    ;;
  debug)
    export MODE="debug"
    ;;
  local)
    export MODE="local"
    ;;
  --help|-h|"")
    usage && exit 0
    ;;
  *)
    printf "Illegal option $1\n"
    usage && exit 1
    ;;
esac

echo DEBUG: MODE is $MODE

if [[ $MODE == "single" ]]; then
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.single.yml up -d
elif [[ $MODE == "e2e" ]]; then
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml up -d
elif [[ $MODE == "debug" ]]; then
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.debug.yml up -d
  while true; do
    read -p "Are you ready? (Y/n) " ANSWER;
      if [[ -z "$ANSWER" ]]; then ANSWER=Y; fi
      case $ANSWER in
          [Yy]* ) break;;
          [Nn]* ) exit 1;;
      esac
  done
elif [[ $MODE == "local" ]]; then
  docker-compose -f docker/docker-compose.common.yml -f docker/docker-compose.yml up -d
  if [[ -f okta-uuid ]]; then
    OKTA_UUID=$(cat okta-uuid)
  fi

  if [[ -z $OKTA_UUID ]]; then
    echo -n "Enter OKTA UUID: "
    read -r OKTA_UUID
    echo $OKTA_UUID > okta-uuid
  fi

  echo DEBUG: OKTA UUID is $OKTA_UUID
fi

pip3 install virtualenv
virtualenv scripts/environment
source scripts/environment/bin/activate
pip3 install -r scripts/requirements.txt
if [[ $MODE == "local" ]]; then
  python3 scripts/start.py --mode $MODE --okta-uuid $OKTA_UUID
else
  python3 scripts/start.py --mode $MODE
fi

deactivate
