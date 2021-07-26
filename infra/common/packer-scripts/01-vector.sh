set -exu

export VECTOR_VERSION=0.13.0
export VECTOR_SHA256=5de3bf7b9a2ef8df733d94edb00ed3c301c900964367d7698b8e7754d02da276

mkdir -p /tmp/build && \
cd /tmp/build && \
wget https://packages.timber.io/vector/${VECTOR_VERSION}/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz && \
echo "${VECTOR_SHA256}  vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" | sha256sum -c - && \
tar xzf vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz --strip-components=2 && \
cp bin/vector /bin && \
cd /tmp && \
rm -rf /tmp/build

addgroup vector && \
adduser -S -G vector vector
setcap cap_dac_override=+eip /bin/vector

# TODO: add sources for nginx when it's installed
cat <<'EOF' > /etc/vector.toml.tpl
[sources.syslog]
type = "file"
ignore_older_secs = 600
include = ["/var/log/messages"]
read_from = "beginning"

[transforms.parse_syslog]
type = "remap"
inputs = ["syslog"]
source = '''
. = parse_syslog!(string!(.message))
'''

[sources.vault]
type = "file"
ignore_older_secs = 600
include = ["/var/log/vault.log"]
read_from = "beginning"

[sinks.gcp]
type = "gcp_stackdriver_logs"
inputs = ["parse_syslog", "vault"]
log_id = "vector-logs"
project_id = "$GCE_PROJECT_ID"
resource.type = "gce_instance"
resource.instance_id = "$GCE_INSTANCE_ID"
resource.zone = "$GCE_ZONE"
EOF

cat <<'EOF' > /etc/vector-config.sh
#!/usr/bin/env bash
export GCE_PROJECT_ID="$(curl -s "http://metadata.google.internal/computeMetadata/v1/project/project-id" -H "Metadata-Flavor: Google" 2>/dev/null)"
export GCE_ZONE="$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/zone" -H "Metadata-Flavor: Google" 2>/dev/null | cut -d/ -f4)"
export GCE_INSTANCE_ID="$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/id" -H "Metadata-Flavor: Google" 2>/dev/null)"
envsubst < /etc/vector.toml.tpl > /etc/vector.toml
EOF

chmod +x /etc/vector-config.sh

cat <<'EOF' > /etc/init.d/vector
#!/sbin/openrc-run

name="Vector logs collector"
description="Vector collecting local logs"
description_reload="Reload configuration"

extra_started_commands="reload"

command="/bin/vector"
command_args="-c /etc/vector.toml"
command_user="vector:vector"

supervisor=supervise-daemon
output_log="/var/log/vector.log"
error_log="/var/log/vector.log"
respawn_max=0
respawn_delay=10

start_pre() {
  checkpath -f -m 0644 -o "$command_user" "$output_log" "$error_log" \
    && checkpath -d -m 0644 -o "$command_user" "/var/lib/vector" \
    && /etc/vector-config.sh
}

reload() {
  start_pre \
    && ebegin "Reloading $RC_SVCNAME configuration" \
    && $supervisor "$RC_SVCNAME" --signal HUP
  eend $?
}
EOF

chmod +x /etc/init.d/vector
