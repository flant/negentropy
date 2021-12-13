#!/usr/bin/env bash

export KAFKA_DOMAIN="$KAFKA_DOMAIN"
export KAFKA_BUCKET="$KAFKA_BUCKET"
export KAFKA_REPLICAS="$KAFKA_REPLICAS"
export SERVER_KEY_PASS="$SERVER_KEY_PASS"

export CERT_VALIDITY_DAYS="7"
export CERT_EXPIRE_SECONDS="172800" # 2 days

export KAFKA_CA_NAME="$KAFKA_CA_NAME"
export KAFKA_CA_POOL="$KAFKA_CA_POOL"
export KAFKA_CA_LOCATION="$KAFKA_CA_LOCATION"

export FQDN="$(hostname).${KAFKA_DOMAIN}"
export NODE_ID="$(hostname | grep -Eo "[0-9]+$")"
export NODE_PREFIX="$(hostname | sed "s/[0-9]*$//")"
export TIMESTAMP="$(date +"%s")"

export KEYSTORE_PATH="/data/keystore"
