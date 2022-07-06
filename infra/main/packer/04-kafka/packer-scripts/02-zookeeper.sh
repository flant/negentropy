set -exu

# TODO: use supervisor=supervise-daemon to restart service on fail
#respawn_max=0
#respawn_delay=10

cat <<'EOF' > /etc/init.d/zookeeper
#!/sbin/openrc-run
name="Zookeeper"
description="Zookeeper for Kafka distributed messaging system"

command="/opt/kafka/bin/zookeeper-server-start.sh"
command_args="/tmp/kafka/zookeeper.properties"
command_user="kafka:kafka"

supervisor=supervise-daemon
output_log="/var/log/zookeeper.log"
error_log="/var/log/zookeeper.log"
respawn_max=0
respawn_delay=10

depend() {
  need net
  after bootmisc
}

start_pre() {
  checkpath -f -m 0644 -o "$command_user" "$output_log" "$error_log" \
    && /bin/update-hostname &> /dev/null \
    && /etc/kafka/scripts/mount-data-disk.sh &> /var/log/mount-data-disk.log \
    && /etc/kafka/scripts/configure-zookeeper.sh \
    && /etc/kafka/scripts/boostrap-stores.sh &> /var/log/boostrap-stores.log \
    && /etc/kafka/scripts/renew-certificate-schedule.sh
}

# Custom start function, because it is necessary to set specific environment variables (e.g. KAFKA_OPTS)
start() {
  ebegin "Starting $name"
  ${supervisor} ${RC_SVCNAME} --start \
    --stdout "$output_log" \
    --stderr "$error_log" \
    --respawn-delay "$respawn_delay" \
    --respawn-max "$respawn_max" \
    --user "$command_user" \
    --env KAFKA_OPTS="-javaagent:/opt/kafka/libs/jmx_prometheus_javaagent.jar=7074:/etc/kafka/zookeeper.metrics.yml" \
    --env KAFKA_HEAP_OPTS="$ZOOKEEPER_HEAP_OPTS" \
    "$command" -- "$command_args"
  eend $?
}
EOF

chmod +x /etc/init.d/zookeeper
rc-update add zookeeper

# Increase ulimit
cat <<'EOF' > /etc/conf.d/zookeeper
rc_ulimit="-n 65536"
EOF
