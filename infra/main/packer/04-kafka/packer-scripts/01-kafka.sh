set -exu

export SCALA_VERSION=2.13
export KAFKA_VERSION=2.8.1
export KAFKA_SHA256=4888b03e3b27dd94f2d830ce3bae9d7d98b0ccee3a5d30c919ccb60e0fa1f139

export PROMETHEUS_JAVAAGENT_VERSION=0.15.0
export PROMETHEUS_JAVAAGENT_SHA256=da1abbb8b553a21ff215a07ff9abf80f0ecb16b63c790b1c296b7959de117e9e

apk add openjdk8-jre

mkdir -p /tmp/build && \
cd /tmp/build && \
wget https://apache-mirror.rbc.ru/pub/apache/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz && \
echo "${KAFKA_SHA256}  kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz" | sha256sum -c - && \
tar xzf kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz && \
mv kafka_${SCALA_VERSION}-${KAFKA_VERSION} /opt/kafka && \
wget https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/${PROMETHEUS_JAVAAGENT_VERSION}/jmx_prometheus_javaagent-${PROMETHEUS_JAVAAGENT_VERSION}.jar && \
echo "${PROMETHEUS_JAVAAGENT_SHA256}  jmx_prometheus_javaagent-${PROMETHEUS_JAVAAGENT_VERSION}.jar" | sha256sum -c - && \
mv jmx_prometheus_javaagent-${PROMETHEUS_JAVAAGENT_VERSION}.jar /opt/kafka/libs/jmx_prometheus_javaagent.jar && \
cd /tmp && \
rm -rf /tmp/build

addgroup kafka && \
adduser -S -G kafka kafka

# Create a mount point for data disk.
mkdir /data
chown kafka:kafka /data

# TODO: use supervisor=supervise-daemon to restart service on fail
#respawn_max=0
#respawn_delay=10

cat <<'EOF' > /etc/init.d/kafka
#!/sbin/openrc-run
name="Kafka broker"
description="Kafka distributed messaging system"

logfile="/var/log/kafka.log"

command="/opt/kafka/bin/kafka-server-start.sh"
command_args="/tmp/kafka/server.properties"

command_background=yes
pidfile=/run/kafka.pid

command_user="kafka:kafka"

start() {
	ebegin "Starting kafka ..."
	start-stop-daemon --start --background --user kafka --group kafka \
	  --chdir /opt/kafka --stdout $logfile --stderr $logfile -m \
    --pidfile $pidfile \
    --env KAFKA_OPTS="-javaagent:/opt/kafka/libs/jmx_prometheus_javaagent.jar=7073:/etc/kafka/server.metrics.yml" \
    --env KAFKA_HEAP_OPTS="$KAFKA_HEAP_OPTS" \
    --env LOG_DIR="/data/logs/kafka" \
    --exec $command -- $command_args
	eend $?
}

depend() {
	after zookeeper
}

start_pre() {
	checkpath -f -m 0644 -o "$command_user" "$logfile" \
    && /bin/update-hostname \
    && /etc/kafka/scripts/configure-kafka.sh
}
EOF

chmod +x /etc/init.d/kafka
rc-update add kafka

# Increase ulimit
cat <<'EOF' > /etc/conf.d/kafka
rc_ulimit="-n 65536"
EOF

