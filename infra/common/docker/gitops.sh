#!/usr/bin/env bash

if [ "$1" == "--debug" ]; then
  docker run --pull always --cap-add=CAP_IPC_LOCK -it -w /app/infra/common/docker --rm -v /tmp/negentropy-gnupg-tmp:/tmp/gnupg -v $(pwd)/../../../:/app alpine:3.15.4 sh -c "apk add -U bash && source ../../env_vars && bash -l"
  exit 0
fi

for i in "$@"; do
  case $i in
    -t=*|--type=*)
      TYPE="${i#*=}"
      shift
      ;;
    -e=*|--env=*)
      ENV="${i#*=}"
      shift
      ;;
    --save-root-tokens-on-initialization)
      SAVE_ROOT_TOKENS="$i"
      shift
      ;;
    --bootstrap)
      BOOTSTRAP="$i"
      shift
      ;;
    -*|--*)
      echo "Unknown option $i"
      exit 1
      ;;
    *)
      ;;
  esac
done

rm -rf /tmp/negentropy-gnupg-tmp
mkdir -p /tmp/negentropy-gnupg-tmp

docker run --pull always --cap-add=CAP_IPC_LOCK -it -w /app --rm -v /tmp/negentropy-gnupg-tmp:/tmp/gnupg -v $(pwd)/../../../:/app alpine:3.15.4 sh -c "cd infra/common/docker && source ../../env_vars && source install-dependencies.sh && ./apply.py --type=$TYPE --env=$ENV $SAVE_ROOT_TOKENS $BOOTSTRAP"
