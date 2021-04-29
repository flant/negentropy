#!/usr/bin/env bash

. /etc/kafka/scripts/variables.sh

mkdir -p /tmp/kafka
envsubst < /etc/kafka/server.properties > /tmp/kafka/server.properties
envsubst < /etc/kafka/client-ssl.properties > /tmp/kafka/client-ssl.properties

zookeeper_connect=""
for ((i=1; i<=$KAFKA_REPLICAS; i++)); do
  zookeeper_connect="$zookeeper_connect,$NODE_PREFIX$i.$MAIN_DOMAIN:2182"
done
echo "zookeeper.connect=${zookeeper_connect:1}" >> /tmp/kafka/server.properties
