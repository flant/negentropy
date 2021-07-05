set -exu

VAULT_VERSION=1.7.0

# We are building our own vault binary and packer should upload it before running this script.
chmod +x /bin/vault

addgroup vault && \
adduser -S -G vault vault

cat <<'EOF' > /etc/vault-config.sh
#!/usr/bin/env bash
export INTERNAL_ADDRESS="$(ip r get 1 | awk '{print $7}')"
envsubst < /etc/vault.hcl > /tmp/vault.hcl
EOF

chmod +x /etc/vault-config.sh

cat <<'EOF' > /etc/init.d/vault
#!/sbin/openrc-run
name="Vault server"
description="Vault is a tool for securely accessing secrets"
description_reload="Reload configuration"

extra_started_commands="reload"

command="/bin/vault"
command_args="server -config /tmp/vault.hcl"
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
	checkpath -f -m 0644 -o "$command_user" "$output_log" "$error_log" \
    && /bin/update-hostname \
    && /etc/vault-config.sh
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

echo "export VAULT_ADDR='http://127.0.0.1:8200'" > /root/.profile
