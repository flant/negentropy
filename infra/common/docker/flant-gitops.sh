#!/usr/bin/env bash

if [ "$1" == "debug" ]; then
  docker run --pull always --cap-add=CAP_IPC_LOCK -it -w /app/infra/common/docker --rm -v /tmp/negentropy-gnupg-tmp:/tmp/gnupg -v $(pwd)/../../../:/app alpine:3.13.5 sh -c "apk add -U bash && source ../../env_vars && bash -l"
  exit 0
fi

rm -rf /tmp/negentropy-gnupg-tmp
mkdir -p /tmp/negentropy-gnupg-tmp
docker run --pull always --cap-add=CAP_IPC_LOCK -it -w /app --rm -v /tmp/negentropy-gnupg-tmp:/tmp/gnupg -v $(pwd)/../../../:/app alpine:3.13.5 sh -c "cd infra/common/docker && source ../../env_vars && source install-dependencies.sh && go version && ./apply.py --save-root-tokens-on-initialization --migration-config $1 --type $2"
