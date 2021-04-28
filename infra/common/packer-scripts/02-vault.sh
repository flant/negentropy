set -exu

VAULT_VERSION=1.7.0

mkdir -p /tmp/build && \
cd /tmp/build && \
wget https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip && \
wget https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_SHA256SUMS && \
grep vault_${VAULT_VERSION}_linux_amd64.zip vault_${VAULT_VERSION}_SHA256SUMS | sha256sum -c && \
unzip -d /bin vault_${VAULT_VERSION}_linux_amd64.zip && \
cd /tmp && \
rm -rf /tmp/build

addgroup vault && \
adduser -S -G vault vault

cat <<'EOF' > /etc/init.d/vault
#!/sbin/openrc-run
name="Vault server"
description="Vault is a tool for securely accessing secrets"
description_reload="Reload configuration"

extra_started_commands="reload"

command="/bin/vault"
command_args="server -config /etc/vault.hcl"
command_user="vault:vault"

supervisor=supervise-daemon
output_log="/var/log/vault.log"
error_log="/var/log/vault.log"
respawn_max=0
respawn_delay=10

depend() {
	need net
	after firewall
}

start_pre() {
	checkpath -f -m 0644 -o "$command_user" "$output_log" "$error_log"
}

reload() {
	start_pre \
		&& ebegin "Reloading $RC_SVCNAME configuration" \
		&& $supervisor "$RC_SVCNAME" --signal HUP
	eend $?
}
EOF

chmod +x /etc/init.d/vault
rc-update add vault

# Add memlock capability
setcap cap_ipc_lock=+ep /bin/vault

# Increase ulimit
cat <<'EOF' > /etc/conf.d/vault
rc_ulimit="-n 65536"
EOF
chmod +x /etc/local.d/ulimit.start && rc-update add local default

echo "export VAULT_ADDR='http://127.0.0.1:8200'" > /root/.profile
