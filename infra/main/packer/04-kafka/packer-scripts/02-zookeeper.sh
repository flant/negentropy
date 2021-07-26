set -exu

# TODO: use supervisor=supervise-daemon to restart service on fail
#respawn_max=0
#respawn_delay=10

cat <<'EOF' > /etc/init.d/zookeeper
#!/sbin/openrc-run
name="Zookeeper"
description="Zookeeper for Kafka distributed messaging system"

logfile="/var/log/zookeeper.log"

command="/opt/kafka/bin/zookeeper-server-start.sh"
command_args="/tmp/kafka/zookeeper.properties"

command_background=yes
pidfile=/run/zookeeper.pid

command_user="kafka:kafka"

start() {
	ebegin "Starting zookeeper ..."
	start-stop-daemon --start --background --user kafka --group kafka \
	  --chdir /opt/kafka --stdout $logfile --stderr $logfile -m \
    --pidfile $pidfile \
    --env KAFKA_OPTS="-javaagent:/opt/kafka/libs/jmx_prometheus_javaagent.jar=7074:/etc/kafka/zookeeper.metrics.yml" \
    --env KAFKA_HEAP_OPTS="$ZOOKEEPER_HEAP_OPTS" \
    --env LOG_DIR="/data/logs/zookeeper" \
    --exec $command -- $command_args
	eend $?
}

depend() {
  need net
	after bootmisc
}

start_pre() {
	checkpath -f -m 0644 -o "$command_user" "$logfile" \
    && /bin/update-hostname \
    && /etc/kafka/scripts/mount-data-disk.sh &> /var/log/kafka-mount-data-disk.log \
    && /etc/kafka/scripts/configure-zookeeper.sh \
    && /etc/kafka/scripts/boostrap-stores.sh &> /var/log/kafka-boostrap-stores.log \
    && /etc/kafka/scripts/renew-certificate-schedule.sh
}
EOF

chmod +x /etc/init.d/zookeeper
rc-update add zookeeper

# Increase ulimit
cat <<'EOF' > /etc/conf.d/zookeeper
rc_ulimit="-n 65536"
EOF
