#!/usr/bin/env bash

. /etc/kafka-variables.sh

mkdir -p /tmp/kafka
envsubst < /etc/kafka/zookeeper.properties > /tmp/kafka/zookeeper.properties

for ((i=1; i<=$KAFKA_REPLICAS; i++)); do
  echo "server.$i=$NODE_PREFIX$i.$KAFKA_DOMAIN:2888:3888" >> /tmp/kafka/zookeeper.properties
done

if [[ ! -f /data/zookeeper/myid ]]; then
    mkdir -p /data/zookeeper
    echo "$NODE_ID" > /data/zookeeper/myid
fi

chown -R kafka:kafka /data/zookeeper
