#!/usr/bin/env bash

# Common variables for all instances.
export HOSTNAME="$(hostname)"

export INTERNAL_ADDRESS="$(ip r get 1 | awk '{print $7}')"

export GCP_PROJECT="$GCP_PROJECT"
export GCP_REGION="$GCP_REGION"
