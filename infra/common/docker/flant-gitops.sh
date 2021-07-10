#!/usr/bin/env bash

rm -rf /tmp/negentropy-gnupg-tmp
mkdir -p /tmp/negentropy-gnupg-tmp
docker run --pull always --cap-add=CAP_IPC_LOCK -it -w /app --rm -v /usr/local/share/ca-certificates/ca.crt:/usr/local/share/ca-certificates/ca.crt -v /tmp/negentropy-gnupg-tmp:/tmp/gnupg -v $(pwd)/../../../:/app alpine:3.13.5 sh -c "cd infra/common/docker && source install-dependencies.sh && source env_vars && go version && update-ca-certificates && ./apply.py --save-root-tokens-on-initialization --type $1"
