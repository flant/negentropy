#!/usr/bin/env bash
set -euo pipefail

cd tests
yarn
yarn test
