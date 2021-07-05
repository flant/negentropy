#!/usr/bin/env bash

rm -rf /tmp/negentropy-gnupg-tmp
mkdir -p /tmp/negentropy-gnupg-tmp
docker run --pull always --cap-add=CAP_IPC_LOCK -it -w /app --rm -v /tmp/negentropy-gnupg-tmp:/tmp/gnupg -v $(pwd)/../../../:/app ubuntu:20.04 bash -c "cd infra/common/docker && source install-dependencies.sh && source env_vars && go version && ./apply.py --save-root-tokens-on-initialization --type $1"
