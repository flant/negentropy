set -exu

# We are building our own vault binary and packer should upload it before running this script.
chmod +x /bin/vault

addgroup vault && \
adduser -S -G vault vault

cat <<'EOF' > /etc/vault-config.sh
#!/usr/bin/env bash
. /etc/vault-variables.sh
envsubst < /etc/vault.hcl > /tmp/vault.hcl
EOF

chmod +x /etc/vault-config.sh

cat <<'EOF' > /etc/vault-init.sh
#!/usr/bin/env bash
. /etc/vault-variables.sh

while true; do
    vault status &>/dev/null
    status=$?
    if [ $status -eq 0 ]; then
      echo "Vault already initialized."
      exit 0
    elif [ $status -eq 2 ]; then
      break
    fi
    echo "Waiting for vault server start, sleeping for 2 seconds..."
    sleep 2
done

echo "Starting vault initialization."
pushd /tmp
until [ -f "${VAULT_ROOT_TOKEN_PGP_KEY}" ]; do
  gsutil cp gs://${TFSTATE_BUCKET}/${VAULT_ROOT_TOKEN_PGP_KEY} .
  sleep 2
done
vault_recovery_pgp_keys="$(find "/etc/recovery-pgp-keys/" -type f | tr '\n' ',' | sed 's/,$//g')"
vault_init_out="$(vault operator init -root-token-pgp-key="${VAULT_ROOT_TOKEN_PGP_KEY}" -recovery-shares="${VAULT_RECOVERY_SHARES}" -recovery-threshold="${VAULT_RECOVERY_THRESHOLD}" -recovery-pgp-keys="${vault_recovery_pgp_keys}")"
echo "$vault_init_out" | grep "^Initial Root Token: " | awk '{print $4}' > "${VAULT_ROOT_TOKEN_ENCRYPTED}"
echo "$vault_init_out" | grep "^Recovery Key " | awk '{print $4}' > "${VAULT_RECOVERY_KEYS_ENCRYPTED}"
gsutil cp {"${VAULT_ROOT_TOKEN_ENCRYPTED}","${VAULT_RECOVERY_KEYS_ENCRYPTED}"} gs://${TFSTATE_BUCKET}/
rm -rf "${VAULT_ROOT_TOKEN_ENCRYPTED}"
rm -rf "${VAULT_RECOVERY_KEYS_ENCRYPTED}"
rm -rf "${VAULT_ROOT_TOKEN_PGP_KEY}"
popd
EOF

chmod +x /etc/vault-init.sh

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

start_post() {
  /etc/vault-init.sh 2>&1 >> /var/log/vault-init.log
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

echo "source /etc/vault-variables.sh" > /root/.profile
