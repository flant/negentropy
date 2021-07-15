#!/usr/bin/env bash

docker run --pull always -it -w /app --rm -v $(pwd)/../../../:/app ubuntu:20.04 bash -c "cd infra/common/docker && source install-dependencies.sh && source env_vars && go version && ./apply.py --type $1"
