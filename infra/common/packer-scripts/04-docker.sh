set -exu

apk update
apk add docker
rc-update add docker boot

# Let Docker to generate key.json and create /opt/containerd.
rc-service docker start && sleep 5 && rc-service docker stop

cat <<'EOF' > /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-file": "1",
    "max-size": "1m"
  },
  "max-concurrent-downloads": 3
}
EOF
